
package main

import "log"
import "fmt"
import "time"
import "net/http"
import "math/rand"
import "github.com/gorilla/websocket"
import "encoding/json"

type user struct {
    name string
    status string // idle, wait, play, watch
    game string
    color string
    opponent string
    conn *websocket.Conn
}

type game struct {
    id string
    white string
    black string
    fen string // "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    watchers []string
}

// http://egypt.silverkeytech.com/blogs/third-world/2013/8/you-cannot-assign-a-struct-field-off-map-directly
var users map[string]*user = map[string]*user{}
var games map[string]*game = map[string]*game{}
var wait string = ""

func main() {
	rand.Seed(time.Now().UnixNano())
	http.Handle("/", http.FileServer(http.Dir("./client")))
	http.HandleFunc("/ws", wsHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

// ******************************************************************
// Gorilla WebSocket

var upgrader = websocket.Upgrader {
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    var connuser *user
    conn, err := upgrader.Upgrade(w, r, nil);

    if err != nil {
        log.Println(err)
        return
    }

    for {

        // get the user name of the connection
        for _, u := range users {
            if u.conn == conn {
                connuser = u
                log.Println("test", connuser)
            }
        }

        _, p, err := conn.ReadMessage();
        if err != nil {
            log.Println("error 1: ", err)
            deleteUser(connuser.name)
            broadcastCounts()
            return
        }
        //log.Println(p)
        //log.Println("pppp ",string(p))

        var jdat map[string]interface{}
        if err := json.Unmarshal(p, &jdat); err != nil {
            log.Println(err)
            return
        }

        //log.Println("=====")
        log.Println("=====",jdat["type"])

        if jdat["type"] == "init" {
            name := newUser()
            users[name].conn = conn
            sendUserInfo(*users[name], "user")
            broadcastCounts();
        }

        if jdat["type"] == "play computer" {
            id := newGame(connuser.name, "computer");
            broadcastCounts();
            broadcastGameInfo(id);
        }

        if jdat["type"] == "play human" {
            if wait == "" {
                wait = connuser.name;
                users[connuser.name].status = "wait";
                sendUserInfo(*users[connuser.name], "wait")
            } else {
                opponent := wait
                wait = ""
                id := newGame(opponent, connuser.name)
                broadcastCounts()
                broadcastGameInfo(id)
            }
        }

        if jdat["type"] == "cancel watch" {
            cancelWatch(connuser.name)
        }

        if jdat["type"] == "cancel wait" {
            if (wait == connuser.name) {
                wait = ""
            }
            users[connuser.name].status = "idle"
            sendUserInfo(*users[connuser.name], "user")
        }

        if jdat["type"] == "watch game" {
            id := gameRand()
            if id != "" {
                games[id].watchers = append(games[id].watchers, connuser.name)
                users[connuser.name].status = "watch"
                users[connuser.name].game = id;
                sendUserInfo(*users[connuser.name], "user")
                p := []byte(fmt.Sprintf(`{"type":"watch game","data":"%s"}`, games[id].fen))
                if err := users[connuser.name].conn.WriteMessage(websocket.TextMessage, p); err != nil {
                    log.Println(err)
                }
                broadcastGameInfo(id);
            }
        }

        if jdat["type"] == "finish game" {
            id := users[connuser.name].game;
            data, _ := jdat["data"].(map[string]interface{})
            if data["draw"] == 1 {
                why = 1
            } else if data["white"] == 1 {
                why = 2
            } else if data["black"] == 1 {
                why = 3
            } else if data["resigned"] == games[id].white {
                why = 4
            } else if data["resigned"] == games[id].black {
                why = 5
            } else if data["disconnected"] == games[id].white {
                why = 6
            } else if data["disconnected"] == games[id].white {
                why = 7
            } else {
                why = 0
            }
            finishGame(id, why);
            broadcastCounts();
        }

        if jdat["type"] == "move" {
            jmove, _ := jdat["data"].(map[string]interface{})
            p := []byte(fmt.Sprintf(`{"type":"move","data":"%s"}`, jmove["move"]))
            var opponent = users[connuser.name].opponent;
            if (opponent != "computer") {
                if err := users[opponent].conn.WriteMessage(websocket.TextMessage, p); err != nil {
                    log.Println(err)
                }
            }
            var id = users[connuser.name].game;
            games[id].fen = jmove["fen"].(string)

            for _, u := range games[id].watchers {
                if err := users[u].conn.WriteMessage(websocket.TextMessage, p); err != nil {
                    log.Println(err)
                }
            }
        }

    }
}

func broadcastCounts() {
    p := []byte(fmt.Sprintf(`{"type":"counts","data":{"usercnt":"%d","gamecnt":"%d"}}`, len(users), len(games)))
    for _, u := range users {
        if err := u.conn.WriteMessage(websocket.TextMessage, p); err != nil {
            log.Println(err)
        }
    }
}

func broadcastGameInfo(id string) {
    p := []byte(fmt.Sprintf(`{"type":"game info","data":{"white":"%s","black":"%s","watchers":"%d"}}`, games[id].white, games[id].black, len(games[id].watchers)))
    // send to white
    if err := users[games[id].white].conn.WriteMessage(websocket.TextMessage, p); err != nil {
        log.Println(err)
    }
    // send to black
    if (games[id].black != "computer") {
        if err := users[games[id].black].conn.WriteMessage(websocket.TextMessage, p); err != nil {
            log.Println(err)
        }
    }
    // send to watchers
    for _, u := range games[id].watchers {
        if err := users[u].conn.WriteMessage(websocket.TextMessage, p); err != nil {
            log.Println(err)
        }
    }
}

func sendUserInfo(u user, t string) {
    p := []byte(fmt.Sprintf(`{"type":"%s","data":{"name":"%s","status":"%s","color":"%s","opponent":"%s"}}`, t, u.name, u.status, u.color, u.opponent))
    if err := u.conn.WriteMessage(websocket.TextMessage, p); err != nil {
        log.Println(err)
        return
    }
}


// generate new user add to users list
func newUser() string {
    // generate non exists name
    name := hexRand();
    _, ok := users[name]
    for ok {
        name = hexRand();
        _, ok = users[name]
    }
    // new user
    var newuser = user{
        name : name,
        status : "idle",    // idle, wait, play, watch
        game : "",
        color : "",
        opponent : "",
        conn : nil,         // connection
    }
    // add to users list
    users[name] = new(user)
    *users[name] = newuser
    return name;
}

func deleteUser(name string) {
    id := users[name].game;
    status := users[name].status
    if (status == "play") {
        why := 6                            // white disconnected
        if users[name].color == "black" {   // black disconnected
            why = 7
        }
        finishGame(id, why);
    } else if (status == "watch") {
        if _, ok := games[id]; ok {
            for i, in := range(games[id].watchers) {
                if in == name {
                    games[id].watchers = append(games[id].watchers[:i], games[id].watchers[i+1:]...)
                }
            }
            broadcastGameInfo(id);
        }
    }
    if (wait == name) {
        wait = "";
    }
    delete(users, name);
}

func newGame(white, black string) string {
    // generate non exists game
    id := hexRand();
    _, ok := games[id]
    for ok {
        id = hexRand();
        _, ok = games[id]
    }
    // new user
    var newgame = game{
        id : id,
        white : white,
        black : black,
        fen : "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
        watchers : []string {},
    }
    games[id] = new(game)
    *games[id] = newgame
    users[white].status = "play";
    users[white].game = newgame.id;
    users[white].color = "white";
    users[white].opponent = black;
    sendUserInfo(*users[white], "start game")
    if (black != "computer") {
        users[black].status = "play";
        users[black].game = newgame.id;
        users[black].color = "black";
        users[black].opponent = white;
        sendUserInfo(*users[black], "start game")
    }
    return id;
}

func cancelWatch(name string) {
    id := users[name].game
    if _, ok := games[id]; ok {
        for i, in := range(games[id].watchers) {
            if in == name {
                games[id].watchers = append(games[id].watchers[:i], games[id].watchers[i+1:]...)
            }
        }
        broadcastGameInfo(id);
    }
    users[name].status = "idle";
    users[name].game = "";
    users[name].color = "";
    users[name].opponent = "";
    sendUserInfo(*users[name], "user")
}


func finishGame(id string, why int) {
    if _, ok := games[id]; !ok {
        return
    }
    // save users info
    white := games[id].white
    black := games[id].black
    watchers := games[id].watchers

    // delete game
    delete(games, id)

    whystr := ""
    if (why == 1) {
        whystr = "Draw"
    } else if (why == 2) {
        whystr = "White ("+ white +") won."
    } else if (why == 3) {
        whystr = "Black ("+ black +") won."
    } else if (why == 4) {
        whystr = "Black ("+ black +") won. " + white + " resigned."
    } else if (why == 5) {
        whystr = "White ("+ white +") won. " + black + " resigned."
    } else if (why == 6) {
        whystr = "Black ("+ black +") won. " + white + " disconnected."
    } else if (why == 7) {
        whystr = "White ("+ white +") won. " + black + " disconnected."
    } else {
        whystr = "Game finished."
    }

    log.Println(whystr)
    p := []byte(fmt.Sprintf(`{"type":"finish game","data":"%s"}`, whystr))

    // do user staff
    if (why != 6) { // if white not disconnected
        users[white].status = "idle";
        users[white].game = "";
        users[white].color = "";
        users[white].opponent = "";
        sendUserInfo(*users[white], "finish user")
        if err := users[white].conn.WriteMessage(websocket.TextMessage, p); err != nil {
            log.Println(err)
            return
        }
    }
    if (black != "computer" && why != 7) { // if black not disconnected
        users[black].status = "idle";
        users[black].game = "";
        users[black].color = "";
        users[black].opponent = "";
        sendUserInfo(*users[black], "finish user")
        if err := users[black].conn.WriteMessage(websocket.TextMessage, p); err != nil {
            log.Println(err)
            return
        }
    }
    for _, name := range watchers {
        if _, ok := users[name]; ok {
            users[name].status = "idle";
            users[name].game = "";
            users[name].color = "";
            users[name].opponent = "";
            sendUserInfo(*users[name], "finish user")
            if err := users[name].conn.WriteMessage(websocket.TextMessage, p); err != nil {
                log.Println(err)
                return
            }
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

