import { Chess } from '/static/js/chess.js';

$(document).ready(function() {
    // === INITIALIZATION ===
    // Get the game data passed from the Go template
    const gameDataEl = $('#game-data');
    const gameId = gameDataEl.data('GameID');
    const playerColor = gameDataEl.data('Color'); // 'white' or 'black' Initialize the chess.js game logic
    const game = new Chess(); 

    // Initialize the chessboard.js UI
    let board = null; // Will be initialized in onReady
    let $status = $('#gameStatus');

    // === CHESSBOARD.JS EVENT HANDLERS ===

    // This function is called when a player starts dragging a piece.
    function onDragStart(source, piece, position, orientation) {
        // 1. Do not pick up pieces if the game is over.
        if (game.isGameOver()) {
            return false;
        }

        // 2. Only pick up pieces for the side to move.
        if ((game.turn() === 'w' && piece.search(/^b/) !== -1) ||
            (game.turn() === 'b' && piece.search(/^w/) !== -1)) {
            return false;
        }

        // 3. Only allow the player to move their own pieces.
        if ((playerColor === 'white' && piece.search(/^b/) !== -1) ||
            (playerColor === 'black' && piece.search(/^w/) !== -1)) {
            return false;
        }
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

        if (game.isCheckmate()) {
            status = `Game over, ${moveColor} is in checkmate.`;
        } else if (game.isDraw()) {
            status = 'Game over, drawn position.';
        } else {
            status = `${moveColor} to move.`;
            if (game.inCheck()) {
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
    updateStatus();

});
