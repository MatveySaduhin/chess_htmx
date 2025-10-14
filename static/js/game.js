console.log('=== GAME.JS STARTING ===');

$(document).ready(function() {
    console.log('1. Document ready');

    const gameDataEl = $('#game-data');
    console.log('2. Game data element found:', gameDataEl.length);
    
    const gameId = gameDataEl.data('gameid');
    const playerColor = gameDataEl.data('color');
    console.log('4. Parsed data - GameID:', gameId, 'Color:', playerColor);

    console.log('5. Chessboard function available:', typeof Chessboard);
    console.log('6. Chess function available:', typeof Chess);

    // Initialize chess.js game logic (global Chess)
    const game = new Chess(); 
    
    // Initialize the chessboard.js UI
    let board = null;
    let $status = $('#gameStatus');

    // === CHESSBOARD.JS EVENT HANDLERS ===
    function onDragStart(source, piece, position, orientation) {
        // 1. Do not pick up pieces if the game is over.
        if (game.game_over()) {
            return false;
        }

        // 2. Only pick up pieces for the side to move.
        if ((game.turn() === 'w' && piece.search(/^b/) !== -1) ||
            (game.turn() === 'b' && piece.search(/^w/) !== -1)) {
            return false;
        }

        // 3. Only allow the player to move their own pieces.
        const isPlayersTurn = (playerColor === 'white' && game.turn() === 'w') || 
                             (playerColor === 'black' && game.turn() === 'b');
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
            promotion: 'q' // NOTE: always promote to a queen for simplicity
        });

        // If the move is illegal, snap the piece back to its original square.
        if (move === null) {
            return 'snapback';
        }
        
        // **IMPORTANT**: If the move is legal, send it to the server.
        // We will implement WebSockets later, for now we log it.
        console.log("Move made:", move);
        // TODO: sendMoveToServer(move);

        updateStatus();
    }

    // This function is called after a piece has been dropped and the animation is complete.
    // It's a good place to update the board position to ensure it's in sync.
    function onSnapEnd() {
        board.position(game.fen());
    }

    // === HELPER FUNCTIONS ===

    // Updates the status text under the board.
    function updateStatus() {
        let status = '';
        const moveColor = (game.turn() === 'b') ? "Black" : "White";

        if (game.in_checkmate()) {
            status = `Game over, ${moveColor} is in checkmate.`;
        } else if (game.in_draw()) {
            status = 'Game over, drawn position.';
        } else {
            status = `${moveColor} to move.`;
            if (game.in_check()) {
                status += ` ${moveColor} is in check.`;
            }
        }
        $status.html(status);
    }
    
    // === BOARD CONFIGURATION ===
    const config = {
        draggable: true,
        position: 'start',
        orientation: playerColor, // This correctly orients the board for the player
        onDragStart: onDragStart,
        onDrop: onDrop,
        onSnapEnd: onSnapEnd,
        pieceTheme: 'https://chessboardjs.com/img/chesspieces/wikipedia/{piece}.png',
    };

    board = Chessboard('myBoard', config);
    console.log('7. Chessboard initialized:', board);
    console.log('=== GAME.JS COMPLETE ===');
    updateStatus();
});
