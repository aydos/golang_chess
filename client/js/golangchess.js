
// chess game variables
var ws, user, game, board, cfgplay, cfgwatch;

// jquery variables
var $username, $usercnt, $gamecnt, $gameinfo;
var $hider, $status, $askplay;
var $turnup, $turndown;
var $playerup, $playerdown;
var $message, $cancel;
var $watchgames;
var $watcherinfo;
var $aydos, $about;

// main function
$(document).ready(function() {
    user = {};
    //game = new Chess();
    cfgplay = {
        draggable: true,
        position: "start",
        onDragStart: onDragStart,
        dropOffBoard: "snapback",
        onDrop: onDrop,
        onSnapEnd: onSnapEnd,
        pieceTheme: "/img/alpha/{piece}.png"
    };
    cfgwatch = {
        position: "start",
        pieceTheme: "/img/alpha/{piece}.png"
    };
    board = new ChessBoard("board", cfgplay);

    $username = $("#username");
    $usercnt = $("#usercnt");
    $gamecnt = $("#gamecnt");
    $gameinfo = $("#gameinfo");
    $hider = $("#hider");
    $status = $("#status");
    $askplay = $("#askplay");
    $turnup = $("#turnup");
    $turndown = $("#turndown");
    $watchgames = $("#watchgames");
    $playerup = $("#playerup");
    $playerdown = $("#playerdown");
    $message = $("#message");
    $cancel = $("#cancel");
    $watcherinfo = $("#watcherinfo");
    $aydos = $("#aydos");
    $about = $("#about");

    $(window).bind("resize", initWindow);
    $(document).bind("keydown", keyDown);

    initWindow();
    clickFunctions();

    if (window["WebSocket"]) {
        ws = new WebSocket("ws://" + window.location.host + "/ws");
        ws.onopen = function() {
            ws.send("init");
        }
        ws.onmessage = function(e) {
            getdata(e.data)
        }
        ws.onclose = function(evt) {
        }
    } else {
        console.log("Your browser does not support WebSockets.");
    }

    askplay();
});

function getdata(data) {

    var m = data.split(",");

    if (m[0] == "user") {           // user info
        user.name = m[1];
        user.status = m[2];
        user.color = m[3];
        user.opponent = m[4];
        $username.html(m[1]);
        if (m[2] == "idle") {
            askplay();
        }
    }

    else if (m[0] == "c") {         // counts
        $usercnt.html(m[1]);
        $gamecnt.html(m[2]);
        if (m[2] == 0) {
            $watchgames.prop("disabled", true);
        } else {
            $watchgames.prop("disabled", false);
        }
    }

    else if (m[0] == "sg") {        // start game
        $hider.fadeOut("fast");
        $status.fadeOut("fast");
        user.name = m[1];
        user.status = m[2];
        user.color = m[3];
        user.opponent = m[4];
        game = new Chess();
        board = new ChessBoard("board", cfgplay);
        board.orientation(user.color);
        if(user.color == "white") {
            $turnup.removeClass("turnwhite").addClass("turnblack");
            $turndown.removeClass("turnblack").addClass("turnwhite");
        } else {
            $turnup.removeClass("turnblack").addClass("turnwhite");
            $turndown.removeClass("turnwhite").addClass("turnblack");
        }
        showTurn();
    }

    else if (m[0] == "wg") {
        $hider.fadeOut("fast");
        $status.fadeOut("fast");
        game = new Chess(m[1]);
        board = new ChessBoard("board", cfgwatch);
        board.position(m[1]);
        board.orientation("white");
        $turnup.removeClass("turnwhite").addClass("turnblack");
        $turndown.removeClass("turnblack").addClass("turnwhite");
        showTurn();
    }

    else if (m[0] == "fu") {  // finish user
        user.name = m[1];
        user.status = m[2];
        user.color = m[3];
        user.opponent = m[4];
    }

    else if (m[0] == "fg") { // finish game
        status = "Game finished."
        if (m[1] == 1) {
            status = "Draw"
        } else if (m[1] == 2) {
            status = "White won"
        } else if (m[1] == 3) {
            status = "Black won"
        } else if (m[1] == 4) {
            status = "White won. Black resigned."
        } else if (m[1] == 5) {
            status = "Black won. White resigned."
        } else if (m[1] == 6) {
            status = "White won. Black disconnected."
        } else if (m[1] == 7) {
            status = "Black won. White disconnected."
        }
        showStatus(status, "OK");
    }

    else if (m[0] == "wait") {
        user.name = m[1];
        user.status = m[2];
        user.color = m[3];
        user.opponent = m[4];
        showStatus("Waiting for opponent", "Cancel");
    }

    else if (m[0] == "gi") {
        if (board.orientation() === "white") {
            $playerup.html(m[2]);
            $playerdown.html(m[1]);
        } else {
            $playerup.html(m[1]);
            $playerdown.html(m[2]);
        }
        if (m[3] > 0) {
            $watcherinfo.html("Watcher this game:<br /><span id='watchercnt'>"+ m[3] + "</span>");
        } else {
            $watcherinfo.html("");
        }
    }

    else if (m[0] == "mv") {
        move = { from: m[1][0]+m[1][1], to: m[1][2]+m[1][3] };
        if (m[1][4]) {
            move.promotion = m[1][4];
        }
        game.move(move);
        board.position(game.fen());
        showTurn();
    }

}

function initWindow() {
    var w = document.body.clientWidth;
    var h = document.body.clientHeight;
    
    var gh = h - $("#menu").outerHeight() - 24;
    var gw = gh + 400;
    $("#game").css({width: gw, height: gh});
    $("#board").css({width: gh, height: gh});
    $("#infoup").css({height: gh});
    $hider.css({width: 2*w, height: 2*h, "margin-left": (0-w), "margin-top": (0-h)});

    board.resize();
}

function keyDown(e) {
    if (e.keyCode == 27) {      // esc
        e.preventDefault();
        if (user.status === "play") {
            if ($status.is(':visible')) { // cancel resign
                $hider.fadeOut("fast");
                $status.fadeOut("fast");
            } else {                           // ask resign
                showStatus("Do you resign the game?", "Yes");
            }
        } else if (user.status === "watch") {  // leave watching game
            ws.send("cw");
        }
    }
}

function showStatus(message, button) {
    $hider.fadeIn("fast");
    $status.fadeIn("fast");
    $message.html(message);
    $cancel.html(button);
}

function showTurn() {
    var turn = game.turn() === "w" ? "white" : "black";
    if (turn === board.orientation()) {
        $turnup.hide();
        $turndown.show();
    } else {
        $turnup.show();
        $turndown.hide();
    }
}

function clickFunctions() {

    $("#btnabout").click(function() {
        $hider.fadeIn("fast");
        $about.fadeIn("fast");
    });

    $("#btnaydos").click(function() {
        $hider.fadeIn("fast");
        $aydos.fadeIn("fast");
    });

    $("#closeabout").click(function() {
        $hider.fadeOut("fast");
        $about.fadeOut("fast");
    });

    $("#closeaydos").click(function() {
        $hider.fadeOut("fast");
        $aydos.fadeOut("fast");
    });

    askplay = function() {
        $hider.fadeIn("fast");
        $askplay.fadeIn("fast");
    };

    $("#playhuman").click(function() {
        $hider.fadeOut("fast");
        $askplay.fadeOut("fast");
        ws.send("ph");
    });

    $("#playcomputer").click(function() {
        $hider.fadeOut("fast");
        $askplay.fadeOut("fast");
        ws.send("pc");
    });

    $("#watchgames").click(function() {
        $hider.fadeOut("fast");
        $askplay.fadeOut("fast");
        ws.send("wg");
    });

    $cancel.click(function() {
        $status.fadeOut("fast");
        $askplay.fadeIn("fast");
        if ($cancel.html() === "Cancel") {
            ws.send("cw");
        }
        if ($cancel.html() === "OK") {
            ;
        }
        if ($cancel.html() === "Yes") {
            if (user.color == "white") {
                status = 4; // white resigned
            } else {
                status = 5; // black resigned
            }
            ws.send("fg,"+status);
        }
    });

}


// ====================================================
// Integration between chessboard.js and chess.js

// do not pick up pieces if the game is over
// only pick up pieces for the side to move
var onDragStart = function(source, piece, position, orientation) {
    if (game.game_over() === true || (game.turn() != piece[0]) ) {
        return false;
    }
    if (game.turn() === "w" && user.color === "black") {
        return false;
    }
    if (game.turn() === "b" && user.color === "white") {
        return false;
    }
};

var onDrop = function(source, target) {
    // see if the move is legal
    var move = game.move({
        from: source,
        to: target,
        promotion: "q"
    });

    move = source + target + "q";
    // illegal move
    if (move === null) return "snapback";

    // make the move
    ws.send("mv," + move + "," + game.fen());

    // if play againts computer
    if (user.opponent === "computer") {
        makeComputerMove();
    }

    updateStatus();
};

// update the board position after the piece snap 
// for castling, en passant, pawn promotion
var onSnapEnd = function() {
    board.position(game.fen());
    showTurn();
};

var updateStatus = function() {
    var status = 0;
    /*var moveColor = 'White';
    if (game.turn() === 'b') {
        moveColor = 'Black';
    }*/
    // checkmate?
    if (game.in_checkmate() === true) {
        if (game.turn() === 'b') {
            status = 2; // white won
        } else {
            status = 3; // black won
        }
        //status = 'Game over, ' + moveColor + ' is in checkmate.';
        ws.send("fg,"+status);
    }
    // draw?
    else if (game.in_draw() === true) {
        status = 1;  // draw
        ws.send("fg,"+status);
    }
    // game still on
    else {
        /*status = moveColor + ' to move';
        // check?
        if (game.in_check() === true) {
            status += ', ' + moveColor + ' is in check';
        }*/
    }
};

var makeComputerMove = function() {
    var possibleMoves = game.moves();

    // game over
    if (possibleMoves.length === 0) return;

    /*
    // Random Move
    var randomIndex = Math.floor(Math.random() * possibleMoves.length);
    game.move(possibleMoves[randomIndex]);
    board.position(game.fen());
    ws.send("mv," + possibleMoves[randomIndex] + "," + fen);
    */

    // GarboChess
    var g_garbo = new Worker("/js/garbochess.min.js");
    g_garbo.onmessage = function (e) {
        move = { from: e.data[0]+e.data[1], to: e.data[2]+e.data[3] };
        if (e.data[4]) {
            move.promotion = e.data[4];
        }
        g_garbo.terminate();
        game.move(move);
        board.position(game.fen());
        showTurn();
        ws.send("mv," + e.data + "," + game.fen());
        updateStatus();
    }
    g_garbo.postMessage("position " + game.fen());
    g_garbo.postMessage("search 800");

};

