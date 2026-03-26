console.log("=== GAME.JS STARTING ===");

$(document).ready(function () {
  const gameDataEl = $("#game-data");
  console.log("2. Game data element found:", gameDataEl.length);

  const gameId = gameDataEl.data("gameid");
  const playerColor = gameDataEl.data("color");
  const mode = gameDataEl.data("mode") || "multiplayer";
  const isSinglePlayer =
    String(gameDataEl.data("is-single-player")) === "true";
  const isOpeningStudy =
    String(gameDataEl.data("is-opening-study")) === "true";
  const isVsComputer =
    String(gameDataEl.data("is-vs-computer")) === "true";
  const initialFEN = gameDataEl.data("initial-fen");

  console.log("4. Parsed data - GameID:", gameId, "Color:", playerColor);
  console.log("5. Mode flags:", {
    mode,
    isSinglePlayer,
    isOpeningStudy,
    isVsComputer,
    initialFEN,
  });

  const game = new Chess();
  if (initialFEN && initialFEN !== "start") {
    try {
      game.load(initialFEN);
    } catch (err) {
      console.warn("Failed to load initial FEN:", err);
    }
  }

  let board = null;
  let $status = $("#gameStatus");
  let ws = null;
  let gameActive = false;
  let boardOrientation = playerColor || "white";
  let engineBusy = false;
  let engineLevel = "easy";

  const $openingName = $("#openingName");
  const $openingEco = $("#openingEco");
  const $openingMoves = $("#openingMoves");
  const $engineStatus = $("#engineStatus");
  const $connectionStatus = $("#connectionStatus");

  function setConnectionStatus(text, ok = true) {
    if ($connectionStatus.length === 0) return;

    $connectionStatus.removeClass(
      "bg-green-100 text-green-800 bg-red-100 text-red-800 bg-gray-100 text-gray-700"
    );

    if (ok) {
      $connectionStatus.addClass("bg-green-100 text-green-800");
      $connectionStatus.html(`
        <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
        <span>${text}</span>
      `);
    } else {
      $connectionStatus.addClass("bg-red-100 text-red-800");
      $connectionStatus.html(`
        <span class="w-2 h-2 bg-red-500 rounded-full"></span>
        <span>${text}</span>
      `);
    }
  }

  function renderOpening(opening) {
    if ($openingName.length === 0) return;

    if (!opening) {
        $openingName.text("Opening explorer unavailable");
        $openingEco.text("");
        $openingMoves.html(
          `<div class="text-gray-400">The upstream explorer service is not responding right now.</div>`
        );
        return;
    }

    $openingName.text(opening.name || "Unknown opening");
    $openingEco.text(opening.eco ? `ECO: ${opening.eco}` : "");

    if (!opening.moves || opening.moves.length === 0) {
      $openingMoves.html(
        `<div class="text-gray-400">No common continuations found.</div>`
      );
      return;
    }

    const html = opening.moves
      .slice(0, 8)
      .map((move) => {
        const total =
          (move.white || 0) + (move.draws || 0) + (move.black || 0);

        return `
          <div class="border rounded-lg p-2">
            <div class="font-medium">${move.san}</div>
            <div class="text-xs text-gray-500">
              Games: ${total} · W ${move.white} / D ${move.draws} / B ${move.black}
            </div>
          </div>
        `;
      })
      .join("");


    $openingMoves.html(html);
  }

    async function surrenderGame() {
      if (!confirm("Are you sure you want to surrender?")) {
        return;
      }
    
      if (isSinglePlayer) {
        try {
          const response = await fetch("/api/game/surrender", {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              game_id: gameId,
            }),
          });
    
          const data = await response.json();
          if (!response.ok) {
            throw new Error(data.error || "Surrender failed");
          }
    
          if (data.fen) {
            game.load(data.fen);
            board.position(data.fen);
          }
    
          gameActive = false;
          $("#myBoard").addClass("board-disabled");
          $status.html(data.result || "You surrendered.");
          $("#surrenderBtn").prop("disabled", true).addClass("opacity-60 cursor-not-allowed");
        } catch (err) {
          console.error("Surrender failed:", err);
          alert(err.message || "Surrender failed");
        }
    
        return;
      }
    
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            type: "surrender",
            payload: "",
          })
        );
      } else {
        alert("Connection lost. Could not surrender.");
      }
    }

  async function refreshOpening() {
    if (!isOpeningStudy) return;

    try {
      const response = await fetch(
        `/api/opening?game_id=${encodeURIComponent(gameId)}`
      );
      if (!response.ok) {
        console.warn("Opening explorer request failed:", response.status);
        return;
      }

      const data = await response.json();
      renderOpening(data.opening);
    } catch (err) {
      console.error("Failed to fetch opening info:", err);
    }
  }

  function connectWebSocket() {
    if (isSinglePlayer) {
      console.log("Skipping WebSocket: single-player mode");
      setConnectionStatus("Single-player mode", true);
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws/game/${gameId}`;

    ws = new WebSocket(wsUrl);

    ws.onopen = function () {
      console.log("WebSocket connected to game:", gameId);
      setConnectionStatus("Connected", true);
    };

    ws.onmessage = function (event) {
      const data = JSON.parse(event.data);
      handleWebSocketMessage(data);
    };

    ws.onclose = function () {
      console.log("WebSocket disconnected");
      setConnectionStatus("Disconnected", false);
    };

    ws.onerror = function (error) {
      console.error("WebSocket error:", error);
      setConnectionStatus("Connection error", false);
    };
  }

  function handleWebSocketMessage(data) {
    console.log("Received WebSocket message:", data);

    switch (data.type) {
      case "move":
        const moveData = data.payload;

        if (moveData.move) {
          game.move(moveData.move);
        } else {
          console.error("No move in payload:", moveData);
        }

        if (moveData.fen) {
          board.position(moveData.fen);
        } else {
          board.position(game.fen());
        }

        updateStatus();
        break;

      case "game_state":
        const gameState = data.payload;
        game.load(gameState.fen);
        board.position(gameState.fen);
        updateStatus();
        console.log("Game state synchronized from FEN");
        break;

      case "error":
        console.error("Server error:", data.payload.error);
        alert("Move error: " + data.payload.error);
        break;

      case "game_start":
        gameActive = true;
        alert("Game starting! Both players connected.");
        document.getElementById("myBoard").classList.remove("board-disabled");
        break;

      case "game_waiting":
        gameActive = false;
        document.getElementById("myBoard").classList.add("board-disabled");
        document.getElementById("gameStatus").innerHTML =
          "Waiting for opponent to connect...";
        break;

      case "player_disconnected":
        gameActive = false;
        document.getElementById("gameStatus").innerHTML =
          "⚠️ Opponent disconnected. Game paused.";
        break;

      case "game_over":
        gameActive = false;
        $("#myBoard").addClass("board-disabled");
      
        if (data.payload && data.payload.fen) {
          game.load(data.payload.fen);
          board.position(data.payload.fen);
        }
      
        $status.html(data.payload?.result || "Game over.");
        $("#surrenderBtn").prop("disabled", true).addClass("opacity-60 cursor-not-allowed");
        break;

      default:
        console.log("Unknown message type:", data.type);
    }
  }

  function sendMoveToServer(move) {
    if (isSinglePlayer) {
      return sendSinglePlayerMove(move);
    }

    if (ws && ws.readyState === WebSocket.OPEN) {
      const message = {
        type: "move",
        payload: move.san,
      };

      ws.send(JSON.stringify(message));
      console.log("Sent move to server:", message);
      console.log("San of the move:", move.san);
    } else {
      console.error("WebSocket not connected");
    }
  }

  async function sendSinglePlayerMove(move) {
    const response = await fetch("/api/single-player/move", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        game_id: gameId,
        move: move.san,
      }),
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || "Move failed");
    }

    if (data.fen) {
      game.load(data.fen);
      board.position(data.fen);
    } else {
      board.position(game.fen());
    }

    if (isOpeningStudy && data.opening) {
      renderOpening(data.opening);
    }

    updateStatus();
    return data;
  }

  async function requestEngineMove() {
    if (!isVsComputer || engineBusy || game.game_over()) return;

    engineBusy = true;
    if ($engineStatus.length) {
      $engineStatus.text("Computer thinking...");
    }

    try {
      const response = await fetch("/api/engine/move", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          game_id: gameId,
          level: engineLevel,
        }),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Engine move failed");
      }

      if (data.fen) {
        game.load(data.fen);
        board.position(data.fen);
      }

      if ($engineStatus.length) {
        $engineStatus.text("Computer moved");
      }

      updateStatus();
    } catch (err) {
      console.error("Engine request failed:", err);
      alert(err.message || "Computer move failed");
    } finally {
      engineBusy = false;
    }
  }

  async function undoSinglePlayerMove() {
    if (!isOpeningStudy) return;

    try {
      const response = await fetch("/api/single-player/undo", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          game_id: gameId,
        }),
      });

      const data = await response.json();
      if (!response.ok) {
        alert(data.error || "Undo failed");
        return;
      }

      if (data.fen) {
        game.load(data.fen);
        board.position(data.fen);
      }

      if (data.opening) {
        renderOpening(data.opening);
      }

      updateStatus();
    } catch (err) {
      console.error("Undo failed:", err);
      alert("Undo failed");
    }
  }

  function onDragStart(source, piece) {
    if (isSinglePlayer) {
      if (game.game_over()) {
        return false;
      }

      if (isOpeningStudy) {
        return true;
      }

      if (
        (game.turn() === "w" && piece.search(/^b/) !== -1) ||
        (game.turn() === "b" && piece.search(/^w/) !== -1)
      ) {
        return false;
      }

      const isPlayersTurn =
        (playerColor === "white" && game.turn() === "w") ||
        (playerColor === "black" && game.turn() === "b");

      if (!isPlayersTurn || engineBusy) {
        return false;
      }

      return true;
    }

    if (!gameActive) {
      console.log("Game not active - can't move");
      return false;
    }

    if (game.game_over()) {
      return false;
    }

    if (
      (game.turn() === "w" && piece.search(/^b/) !== -1) ||
      (game.turn() === "b" && piece.search(/^w/) !== -1)
    ) {
      return false;
    }

    const isPlayersTurn =
      (playerColor === "white" && game.turn() === "w") ||
      (playerColor === "black" && game.turn() === "b");
    if (!isPlayersTurn) {
      return false;
    }

    return true;
  }

  function onDrop(source, target) {
    let move = game.move({
      from: source,
      to: target,
      promotion: "q",
    });

    if (move === null) {
      return "snapback";
    }

    console.log("Move made:", move);

    if (isSinglePlayer) {
      sendSinglePlayerMove(move)
        .then(() => {
          if (isVsComputer && !game.game_over()) {
            const computerTurn =
              (playerColor === "white" && game.turn() === "b") ||
              (playerColor === "black" && game.turn() === "w");

            if (computerTurn) {
              requestEngineMove();
            }
          } else if (isOpeningStudy) {
            refreshOpening();
          }
        })
        .catch((err) => {
          console.error("Single-player move failed:", err);
          alert(err.message || "Move failed");
          location.reload();
        });

      updateStatus();
      return true;
    }

    sendMoveToServer(move);
    updateStatus();

    return true;
  }

  function onSnapEnd() {
    board.position(game.fen());
  }

  function updateStatus() {
    let status = "";
    const moveColor = game.turn() === "b" ? "Black" : "White";

    if (game.in_checkmate()) {
      status = `Game over, ${moveColor} is in checkmate.`;
    } else if (game.in_draw()) {
      status = "Game over, drawn position.";
    } else if (isOpeningStudy) {
      status = `Opening study mode — ${moveColor} to move.`;
      if (game.in_check()) {
        status += ` ${moveColor} is in check.`;
      }
    } else if (isVsComputer) {
      const isPlayersTurn =
        (playerColor === "white" && game.turn() === "w") ||
        (playerColor === "black" && game.turn() === "b");

      status = isPlayersTurn ? "Your move." : "Computer thinking...";
      if (game.in_check()) {
        status += ` ${moveColor} is in check.`;
      }
    } else {
      status = `${moveColor} to move.`;
      if (game.in_check()) {
        status += ` ${moveColor} is in check.`;
      }
    }

    $status.html(status);
  }

  const config = {
    draggable: true,
    position: initialFEN && initialFEN !== "start" ? initialFEN : "start",
    orientation: boardOrientation,
    onDragStart: onDragStart,
    onDrop: onDrop,
    onSnapEnd: onSnapEnd,
    pieceTheme:
      "https://raw.githubusercontent.com/lichess-org/lila/master/public/piece/maestro/{piece}.svg",
  };

  board = Chessboard("myBoard", config);
  console.log("7. Chessboard initialized:", board);

  if (isSinglePlayer) {
    gameActive = true;
    document.getElementById("myBoard").classList.remove("board-disabled");
  }

  connectWebSocket();

  $("#surrenderBtn").on("click", function () {
    surrenderGame();
  });

  $("#undoBtn").on("click", function () {
    if (!isOpeningStudy) {
      alert("Undo is only available in opening study.");
      return;
    }
    undoSinglePlayerMove();
  });

  $("#flipBtn").on("click", function () {
    boardOrientation = boardOrientation === "white" ? "black" : "white";
    board.orientation(boardOrientation);
  });

  if (isOpeningStudy) {
    refreshOpening();
  }

  if (
    isVsComputer &&
    playerColor === "black" &&
    game.turn() === "w" &&
    !game.game_over()
  ) {
    setTimeout(() => requestEngineMove(), 250);
  }

  console.log("=== game.js complete ===");
  updateStatus();
});
