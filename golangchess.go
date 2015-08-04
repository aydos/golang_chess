
package main

import (
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "time"
    "strings"
    ws "github.com/gorilla/websocket"
)

type user struct {
    name string
    status string // idle, wait, play, watch
    game string
    color string
    opponent *ws.Conn
}

type game struct {
    id string
    white *ws.Conn
    black *ws.Conn
    fen string // "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    watchers map[*ws.Conn]bool
}

// http://egypt.silverkeytech.com/blogs/third-world/2013/8/you-cannot-assign-a-struct-field-off-map-directly
var users map[*ws.Conn]*user = make(map[*ws.Conn]*user)
var games map[string]*game = make(map[string]*game)
var wait *ws.Conn = nil

func main() {
    log.Println("Golang-Chess'e Hoşgeldiniz....")
	rand.Seed(time.Now().UnixNano())
	http.Handle("/", http.FileServer(http.Dir("./client")))
	http.HandleFunc("/ws", wsHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

// ******************************************************************
// Gorilla WebSocket

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := ws.Upgrade(w, r, nil, 1024, 1024);

    if _, ok := err.(ws.HandshakeError); ok {
        http.Error(w, "Not a websocket handshake", 400)
        return
    } else if err != nil {
        log.Println(err)
        return
    }

    defer conn.Close()

    users[conn] = newUser()

    for {

        // read message
        _, p, err := conn.ReadMessage();
        if err != nil {
            log.Println(err)
            deleteUser(conn)
            broadcastCounts()
            return
        }

        // split message into array
        m := strings.Split(string(p), ",")

        if m[0] == "init" {
            sendUserInfo(conn, "user")
            broadcastCounts();
        }

        if m[0] == "pc" { // play computer
            id := newGame(conn, nil);
            broadcastCounts();
            broadcastGameInfo(id);
        }

        if m[0] == "ph" { // play human
            if wait == nil {
                wait = conn;
                users[conn].status = "wait";
                sendUserInfo(conn, "wait")
            } else {
                opponent := wait
                wait = nil
                id := newGame(opponent, conn)
                broadcastCounts()
                broadcastGameInfo(id)
            }
        }

        if m[0] == "cg" { // cancel watch game
            cancelWatch(conn)
        }

        if m[0] == "cw" { // cancel wait
            if (wait == conn) {
                wait = nil
            }
            users[conn].status = "idle"
            sendUserInfo(conn, "user")
        }

        if m[0] == "wg" { // watch game
            id := gameRand()
            if id != "" {
                games[id].watchers[conn] = true
                users[conn].status = "watch"
                users[conn].game = id;
                sendUserInfo(conn, "user")
                p := []byte(fmt.Sprintf("wg,%s", games[id].fen))
                if err := conn.WriteMessage(ws.TextMessage, p); err != nil {
                    log.Println(err)
                }
                broadcastGameInfo(id);
            }
        }

        if m[0] == "fg" { // finish game
            id := users[conn].game
            finishGame(id, m[1])
            broadcastCounts()
        }

        if m[0] == "mv" { // move
            // m[1] : move
            // m[2] : fen
            p := []byte(fmt.Sprintf("mv,%s", m[1]))
            var opponent = users[conn].opponent;
            if (opponent != nil) {
                if err := opponent.WriteMessage(ws.TextMessage, p); err != nil {
                    log.Println(err)
                }
            }
            var id = users[conn].game;
            games[id].fen = m[2]
            for c := range games[id].watchers {
                if err := c.WriteMessage(ws.TextMessage, p); err != nil {
                    log.Println(err)
                }
            }
        }

    }
}

func broadcastCounts() {
    p := []byte(fmt.Sprintf("c,%d,%d", len(users), len(games)))
    for conn := range users {
        if err := conn.WriteMessage(ws.TextMessage, p); err != nil {
            log.Println(err)
        }
    }
}

func broadcastGameInfo(id string) {
    whitename := users[games[id].white].name
    blackname := ""
    if (games[id].black == nil) {
        blackname = "computer"
    } else {
        blackname = users[games[id].black].name
    }
    p := []byte(fmt.Sprintf("gi,%s,%s,%d", whitename, blackname, len(games[id].watchers)))
    // send to white
    if err := games[id].white.WriteMessage(ws.TextMessage, p); err != nil {
        log.Println(err)
    }
    // send to black
    if (games[id].black != nil) {
        if err := games[id].black.WriteMessage(ws.TextMessage, p); err != nil {
            log.Println(err)
        }
    }
    // send to watchers
    for c := range games[id].watchers {
        if err := c.WriteMessage(ws.TextMessage, p); err != nil {
            log.Println(err)
        }
    }
}

func sendUserInfo(conn *ws.Conn, t string) {
    opponent := ""
    if users[conn].game != "" {
        if users[conn].opponent == nil {
            opponent = "computer"
        } else {
            opponent = users[users[conn].opponent].name
        }
    }
    p := []byte(fmt.Sprintf("%s,%s,%s,%s,%s", t, users[conn].name, users[conn].status, users[conn].color, opponent))
    if err := conn.WriteMessage(ws.TextMessage, p); err != nil {
        log.Println(err)
        return
    }
}

// generate new user add to users list
func newUser() *user {
   // new user, name may not be unique
    var newuser = user {
        name : hexRand(),
        status : "idle",    // idle, wait, play, watch
        game : "",
        color : "",
        opponent : nil,
    }
    return &newuser;
}

func deleteUser(conn *ws.Conn) {
    id := users[conn].game;
    status := users[conn].status
    if (status == "play") {
        why := "6"                          // white disconnected
        if users[conn].color == "black" {   // black disconnected
            why = "7"
        }
        finishGame(id, why);
    } else if (status == "watch") {
        if _, ok := games[id]; ok {
            delete(games[id].watchers, conn)
            broadcastGameInfo(id);
        }
    }
    if (wait == conn) {
        wait = nil;
    }
    delete(users, conn);
}

func newGame(white, black *ws.Conn) string {
    // generate non exists game
    id := hexRand();
    _, ok := games[id]
    for ok {
        id = hexRand();
        _, ok = games[id]
    }
    // new game
    var newgame = game{
        id : id,
        white : white,
        black : black,
        fen : "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
        watchers : map[*ws.Conn]bool{},
    }
    games[id] = new(game)
    *games[id] = newgame
    users[white].status = "play";
    users[white].game = newgame.id;
    users[white].color = "white";
    users[white].opponent = black;
    sendUserInfo(white, "sg")
    if (black != nil) {
        users[black].status = "play"
        users[black].game = newgame.id;
        users[black].color = "black"
        users[black].opponent = white
        sendUserInfo(black, "sg")
    }
    return id;
}

func cancelWatch(conn *ws.Conn) {
    id := users[conn].game
    if _, ok := games[id]; ok {
        delete(games[id].watchers, conn)
        broadcastGameInfo(id)
    }
    users[conn].status = "idle"
    users[conn].game = ""
    users[conn].color = ""
    users[conn].opponent = nil
    sendUserInfo(conn, "user")
}


func finishGame(id, why string) {
    if _, ok := games[id]; !ok {
        return
    }
    // save users info
    white := games[id].white
    black := games[id].black
    watchers := games[id].watchers

    // delete game
    delete(games, id)

    p := []byte(fmt.Sprintf("fg,%s", why))

    // do user staff
    if (why != "6") { // if white not disconnected
        users[white].status = "idle"
        users[white].game = ""
        users[white].color = ""
        users[white].opponent = nil
        sendUserInfo(white, "fu")
        if err := white.WriteMessage(ws.TextMessage, p); err != nil {
            log.Println(err)
            return
        }
    }
    if (black != nil && why != "7") { // if black not disconnected
        users[black].status = "idle"
        users[black].game = ""
        users[black].color = ""
        users[black].opponent = nil
        sendUserInfo(black, "fu")
        if err := black.WriteMessage(ws.TextMessage, p); err != nil {
            log.Println(err)
            return
        }
    }
    for c := range watchers {
        users[c].status = "idle"
        users[c].game = ""
        users[c].color = ""
        users[c].opponent = nil
        sendUserInfo(c, "fu")
        if err := c.WriteMessage(ws.TextMessage, p); err != nil {
            log.Println(err)
            return
        }
    }
}


// ******************************************************************
// Random generators

func Rand(min, max int) int {
	return rand.Intn(max - min + 1) + min;
}

func hexRand() string {
    var hex = make([]byte, 8)
    var chrs = "0123456789ABCDEF"
    for i:=0; i<8; i++ {
        hex[i] = chrs[Rand(0,15)];
    }
    return string(hex);
}

func gameRand() string {
    if len(games) <= 0 {
        return ""
    }
    ind := Rand(1, len(games))
    i := 1
    for g := range games {
        if i == ind {
            return g
        }
        i++
    }
    return ""
}


