console.log("=== GAME.JS STARTING ===");

$(document).ready(function () {
  const gameDataEl = $("#game-data");
  console.log("2. Game data element found:", gameDataEl.length);

  const gameId = gameDataEl.data("gameid");
  const playerColor = gameDataEl.data("color");
  console.log("4. Parsed data - GameID:", gameId, "Color:", playerColor);

  // Initialize chess.js game logic (global Chess)
  const game = new Chess();

  // Initialize the chessboard.js UI
  let board = null;
  let $status = $("#gameStatus");
  let ws = null;

  function connectWebSocket() {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws/game/${gameId}`;

    ws = new WebSocket(wsUrl);

    ws.onopen = function () {
      console.log("WebSocket connected to game:", gameId);
    };

    ws.onmessage = function (event) {
      const data = JSON.parse(event.data);
      handleWebSocketMessage(data);
    };

    ws.onclose = function () {
      console.log("WebSocket disconnected");
    };

    ws.onerror = function (error) {
      console.error("WebSocket error:", error);
    };
  }

  let gameActive = false;

  function handleWebSocketMessage(data) {
    console.log("Received WebSocket message:", data);

    switch (data.type) {
      case "move":
        // Apply opponent's move using SAN
        const moveData = data.payload;

        if (moveData.move) {
          // Use the SAN move directly
          game.move(moveData.move);
        } else {
          console.error("No move in payload:", moveData);
        }

        // Update board with new FEN from server
        if (moveData.fen) {
          board.position(moveData.fen);
        } else {
          board.position(game.fen());
        }

        updateStatus();
        break;

      case "game_state":
        // FEN contains everything needed
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
        // Update UI to show game is active
        break;

      case "game_waiting":
        gameActive = false;
        document.getElementById("myBoard").classList.add("board-disabled");
        document.getElementById("gameStatus").innerHTML =
          "Waiting for opponent to connect...";
        board.draggable = false; // Disable moves
        break;

      case "player_disconnected":
        gameActive = false;
        document.getElementById("gameStatus").innerHTML =
          "⚠️ Opponent disconnected. Game paused.";
        board.draggable = false; // Disable moves
        break;

      default:
        console.log("Unknown message type:", data.type);
    }
  }

  function sendMoveToServer(move) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      const moveData = {
        from: move.from,
        to: move.to,
        flags: move.flags || "",
      };

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
  // === CHESSBOARD.JS EVENT HANDLERS ===
  function onDragStart(source, piece, position, orientation) {
    // 0. Check if the game is active
    if (!gameActive) {
      console.log("Game not active - can't move");
      return false;
    }

    // 1. Do not pick up pieces if the game is over.
    if (game.game_over()) {
      return false;
    }

    // 2. Only pick up pieces for the side to move.
    if (
      (game.turn() === "w" && piece.search(/^b/) !== -1) ||
      (game.turn() === "b" && piece.search(/^w/) !== -1)
    ) {
      return false;
    }

    // 3. Only allow the player to move their own pieces.
    const isPlayersTurn =
      (playerColor === "white" && game.turn() === "w") ||
      (playerColor === "black" && game.turn() === "b");
    if (!isPlayersTurn) {
      return false;
    }

    return true;
  }

  // This function is called when a player drops a piece.
  function onDrop(source, target) {
    // Attempt to make the move in the chess.js engine.
    let move = game.move({
      from: source,
      to: target,
      promotion: "q", // NOTE: always promote to a queen for simplicity
    });

    // If the move is illegal, snap the piece back to its original square.
    if (move === null) {
      return "snapback";
    }

    console.log("Move made:", move);

    sendMoveToServer(move);
    updateStatus();

    return true;
  }

  // This function is called after a piece has been dropped and the animation is complete.
  // It's a good place to update the board position to ensure it's in sync.
  function onSnapEnd() {
    board.position(game.fen());
  }

  // === HELPER FUNCTIONS ===

  // Updates the status text under the board.
  function updateStatus() {
    let status = "";
    const moveColor = game.turn() === "b" ? "Black" : "White";

    if (game.in_checkmate()) {
      status = `Game over, ${moveColor} is in checkmate.`;
    } else if (game.in_draw()) {
      status = "Game over, drawn position.";
    } else {
      status = `${moveColor} to move.`;
      if (game.in_check()) {
        status += ` ${moveColor} is in check.`;
      }
    }
    $status.html(status);
  }
  //
  // === BOARD CONFIGURATION ===
  const config = {
    draggable: true,
    position: "start",
    orientation: playerColor, // This correctly orients the board for the player
    onDragStart: onDragStart,
    onDrop: onDrop,
    onSnapEnd: onSnapEnd,
    pieceTheme:
      "https://raw.githubusercontent.com/lichess-org/lila/master/public/piece/maestro/{piece}.svg",
  };

  board = Chessboard("myBoard", config);
  console.log("7. Chessboard initialized:", board);

  connectWebSocket();

  console.log("=== game.js complete ===");
  updateStatus();
});
