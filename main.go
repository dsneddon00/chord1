package main

import (
  "log"
  "net"
  "net/http"
  "net/rpc"
  "flag"
  "os"
  "bufio"
  "strings"
  "fmt"
  "math/big"
  "crypto/sha1"
)

const (
  defaultHost = "localhost"
  defaultPort = ":3410"
  keySize = sha1.Size * 8
)

var two = big.NewInt(2)
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(keySize), nil)
var allCommands []string

// server is on
var serverOnline = false

type Feed struct {
  Messages []string
}

type handler func(*Node)
type Server chan<- handler
type Nothing struct{}

type Node struct {
  Address string
  Indentifier *big.Int
  // for a ring that only has one Node, its successor will be itself.
  Successor []string
  Predecessor string
  // we have to ensure that this map is working
  Data map[string]string
}

func initCommands() {
  allCommands = append(allCommands, "Help")
  allCommands = append(allCommands, "Port")
  allCommands = append(allCommands, "Quit")
  allCommands = append(allCommands, "Create")
  allCommands = append(allCommands, "Ping")
  allCommands = append(allCommands, "Join")
  allCommands = append(allCommands, "Dump")
  allCommands = append(allCommands, "Put")
  allCommands = append(allCommands, "Get")
  allCommands = append(allCommands, "Delete")
}

func getLocalAddress() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP.String()
}

// Post uneeded for the assignment
/*
// Post method based on the feed pointer, takes in a string parameter and returns nothing
func (s Server) Post(msg string, reply *Nothing) error {
  finished := make(chan struct{})
  s <- func (f *Node) {
    f.Messages = append(f.Messages, msg)
    finished <- struct{}{}
  }
  // what the actual fuck does <-finished mean? It's not even sending it to anything
  <-finished
  return nil
}
*/

/*
// count is the number of messages to retrieve
func (s Server) Get(count int, reply *[]string) error {
  finished := make(chan struct{})
  s <- func(f *Node) {
    // resets count if count is greater than the total ammount of messages
    if len(f.Messages) < count {
      count = len(f.Messages)
    }
    *reply = make([]string, count)
    copy(*reply, f.Messages[len(f.Messages)-count:])
    finished <- struct{}{}
  }
  <-finished
  return nil
  // nil indicates the success
}
*/

func startActor(node *Node) Server {

  ch := make(chan handler)
  //state := new(Feed)
  state := node


  go func() {
    //fmt.Printf("We got here inside the goroutine!!\n")
    for f := range ch {
      //fmt.Printf("We got here inside the loop!!\n")
      // the fix for the panic error was that I was passing state in as a reference rather than as a value, which needed to be changed.
      // error: f(&state)
      f(state)
    }
  }()
  return ch
}

func main() {
  initCommands()
  fmt.Println("Your current address is: " + getLocalAddress() )
  shell(defaultHost+defaultPort)



  /* old code, apparently you start as a client that creates a server ring, you don't start the ring by default
  var isServer bool
  var isClient bool
  var address string
  flag.BoolVar(&isServer, "server", false, "start as tweeter server")
  flag.BoolVar(&isClient, "client", false, "start as tweeter client")
  flag.Parse()

  if isServer && isClient {
    log.Fatalf("cannot be both a client and a server")
  }
  if !isServer && !isClient {
    printUsage()
  }

  switch flag.NArg() {
  case 0:
    if isClient {
      address = defaultHost + ":" + defaultPort
    } else {
      address = ":" + defaultPort
    }
  case 1:
    //user specified the address
    address = flag.Arg(0)
  default:
    printUsage()

  }

  if isClient {
    shell(address)
  } else {
    server(address)
  }

  //client()

  // other old code
  // test cases
  state := new(Feed)

  var trash Nothing
  // capture the error and if it isn't nil then it's a problem
  if err := state.Post("Hello, world!", &trash); err != nil {
    log.Fatalf("Post: %v", err)
  }
  if err := state.Post("Today is Monday", &trash); err != nil {
    log.Fatalf("Post: %v", err)
  }

  var lst []string
  if err := state.Get(5, &lst); err != nil {
    log.Fatalf("Get: %v", err)
  }
  for _, elt := range lst {
    log.Println(elt)
  }
  */

}

func printUsage() {
  log.Printf("Usage %s [-server or -client] [address]", os.Args[0])
  flag.PrintDefaults()
  os.Exit(1)
}

func server(address string, node *Node) {
  actor := startActor(node)
  rpc.Register(actor)
  rpc.HandleHTTP()
  l, e := net.Listen("tcp", address)
  if e != nil {
    log.Fatal("listen error: ", e)
  }
  if err := http.Serve(l, nil); err != nil {
    log.Fatalf("http.Server: %v", err)
  }
}

func client(address string) {

  var trash Nothing
  if err := call(address, "Server.Post", "Hello, again", &trash); err != nil {
    log.Fatalf("client.Call: %v", err)
  }
  if err := call(address, "Server.Post", "I ate cereal for breakfast", &trash); err != nil {
    log.Fatalf("client.Call: %v", err)
  }

  var lst [] string
  if err := call(address, "Server.Get", 5, &lst); err != nil {
    log.Fatalf("client.Call Get: %v", err)
  }

  for _, elt := range lst {
    log.Println(elt)
  }
}

func call(address string, method string, request interface{}, response interface{}) error {
  client, err := rpc.DialHTTP("tcp", address)
  if err != nil {
    log.Printf("rpc.DialHTTP: %v", err)
    return err
  }
  defer client.Close()

  if err := client.Call(method, request, response); err != nil {
    log.Printf("client.Call %s: %v", method, err)
    return err
  }
  return nil
}

func shell(address string) {
  log.Printf("Starting interactive shell")
  //log.Printf("Commands are: get, post")
  var node = Node{}

  scanner := bufio.NewScanner(os.Stdin)
  for scanner.Scan() {
    line := scanner.Text()
    line = strings.TrimSpace(line)

    parts := strings.Split(line, " ")
    if len(parts) > 1 {
      parts[1] = strings.TrimSpace(parts[1])
    }

    if len(parts) == 0 {
      continue
    }

    switch parts[0] {
      case "help":
        if serverOnline == true {
          var trash = Nothing{}
          var commands []string
          if err := call(address, "Server.Help", &trash, &commands); err != nil {
            log.Fatalf("Server calling Server.Help: %v", err)
          }
        }
        // _ ignores a value in a for loop
        for _, elt := range allCommands {
          fmt.Printf(elt + " ")
        }
        fmt.Printf("\n")

      case "port":
        if serverOnline {
          log.Printf("No")
        } else {
          if len(parts) != 2 {
            log.Printf("Invalid command. Try again.\n")
            continue
          }
          address = getLocalAddress() + ":" + parts[1]
          log.Printf("Address: " + address)
        }

      case "quit":
        os.Exit(1)


      case "create":
        // default successor list size is 5
        var successors = make([]string, 5)
        for i := 0; i < 5; i++ {
          successors = append(successors, address)
        }
        var id = hashString(address)
        var data = make(map[string]string)
        // "" is nil
        node = Node{Address: address, Indentifier: id, Successor: successors, Predecessor: "", Data: data}
        serverOnline = true
        go server(address, &node)
        // Implement a skeleton create function that starts a Node instance complete with RPC server. Start by having it listen for a ping method that just responds to all requests immediately.

      case "ping":
        if serverOnline == true {
          // command line args
          if len(parts) != 2 {
            log.Printf("Not specified")
            continue
          }
          var trash = Nothing{}
          // attempts to call the ping user command
          // the two trashes are sending and recieving nothing simply due to testing the conntection
          if err := call(parts[1], "Server.Ping", &trash, &trash); err != nil {
            continue
          } else {
            log.Printf("success at %v", parts[1])
          }
        } else {
          log.Printf("Turn the server on first.")
        }
      case "put":
        if serverOnline == true {
          if len(parts) != 4 {
            log.Printf("Review you command line arguments")
            continue
          }
          var trash = Nothing{}
          // parts[3] is the address
          if err := call(parts[3], "Server.Put", parts[1:3], &trash); err != nil {
            continue
          } else {
            log.Printf("inserted key %v value %v into node at address %v", parts[1], parts[2], node.Address)
          }
        } else {
          log.Printf("Turn the server on first.")
        }

      case "get":
        if serverOnline == true {
          if len(parts) != 3 {
            log.Printf("Review you command line arguments")
            continue
          }
          var trash = ""
          if err := call(parts[2], "Server.Get", parts[1], &trash); err != nil {
            continue
          } else {
            log.Printf("Retrieved %v", trash)
          }
        } else {
          log.Printf("Turn the server on first.")
        }

      case "delete":
        if serverOnline == true {
          if len(parts) != 3 {
            log.Printf("only %v out of 3 arguments", len(parts))
            continue
          }
          var trash = Nothing{}
          if err := call(parts[2], "Server.Delete", parts[1], &trash); err != nil {
            continue
          } else {
            log.Printf("Deleted key %v from %v", parts[1], parts[2])
          }
        } else {
          log.Printf("Turn the server on first.")
        }

      case "dump":
        if serverOnline == true {
          // print of basic info
          log.Printf("Address %v\nIndentifier: %v", node.Address, node.Indentifier)
          log.Printf("Successors: ")
          for i, s := range node.Successor {
            log.Printf("%v: %v", i, s)
          }
          log.Printf("Predecessor: %v", node.Predecessor)
          for k, v := range node.Data {
            log.Printf(k + ": " + v)
          }
        } else {
          log.Printf("Turn the server on first.")
        }

      case "join":
        if serverOnline == true {
          log.Printf("already online, can't make a new ring if there already is one")
        } else {
          // default 5 as the second parameter, it is the list size
          var successors = make([]string, 5)
          var trash = Nothing{}
          var id = hashString(address)
          var data = make(map[string]string)
          if err := call(parts[1], "Server.Join", node.Address, &trash); err != nil {
            continue
          } else {
            var successor = parts[1]
            for i := 0; i < 5; i++ {
              successors[i] = successor
            }
            log.Printf("Joined a new ring with the node address %v", parts[1])
          }
          node = Node{Address: address, Indentifier: id, Successor: successors, Predecessor: "", Data: data}
          serverOnline = true
          go server(address, &node)
        }
      default:
        log.Printf("I only recognize \"get\" and \"post\"")
      }



    }

/*
    switch parts[0] {
    case "help":
      var trash Nothing
      var commands []string
      log.Println("help has been called")
      if err := call(address, "Server.Help", &trash, &commands); err != nil {
        log.Fatalf("call Server.Help: %v", err)
      }
      for _, elt := range commands {
        log.Println(elt)
      }
    /*
    case "get":
      n := 10
      if len(parts) == 2 {
        var err error
        if n, err = strconv.Atoi(parts[1]); err != nil {
          log.Fatalf("parsing number of messages: %v", err)
        }
      }


      var messages [] string
      if err := call(address, "Server.Get", n, &messages); err != nil {
        log.Fatalf("calling Server.Get: %v", err)
      }
      for _, elt := range messages {
        log.Println(elt)
      }

    case "post":
      if len(parts) != 2 {
        log.Printf("you must specify a message to post")
        continue
      }

      var junk Nothing
      if err := call (address, "Server.Post", parts[1], &junk); err != nil {
        log.Fatalf("calling Server.Post: %v", err)
      }

    default:
      log.Printf("I only recognize \"get\" and \"post\"")
    }
*/

    if err := scanner.Err(); err != nil {
      log.Fatalf("scanner error: %v", err)
  }
}

// help command
func (s Server) Help(trash *Nothing, reply *[]string) error {
  finished := make(chan struct{})
  s <- func(n *Node) {
    *reply = make([]string, len(allCommands))
    copy(*reply, allCommands)
    finished <- struct{}{}
  }
  <-finished
  return nil
}

// format taken from tweeter
// for testing use ping localhost:3410
func (s Server) Ping(msg *Nothing, reply *Nothing) error {
  finished := make(chan struct{})
  s <- func(n *Node) {
    finished <- struct{}{}
  }
  <-finished
  return nil
}

// main difference is the data line where we store a value to a key
func (s Server) Put(kv []string, reply *Nothing) error {
	finished := make(chan struct{})
	s <- func(n *Node) {
		finished <- struct{}{}
		n.Data[kv[0]] = kv[1]
	}
	<-finished
	return nil
}

func (s Server) Get(msg string, reply *string) error {
  finished := make(chan struct{})
  s <- func(n *Node) {
    finished <- struct{}{}
    if v, exists := n.Data[msg]; exists {
      *reply = v
    }
  }
  <-finished
  return nil
}

func (s Server) Delete(key string, reply *Nothing) error {
  finished := make(chan struct{})
  s <- func (n *Node) {
    if _, exists := n.Data[key]; exists {
      delete(n.Data, key)
    }
    finished <- struct{}{}
  }
  <-finished
  return nil
}

func (s Server) Join(addr string, reply *Nothing) error {
  finished := make(chan struct{})
  s <- func(n *Node) {
    finished <- struct{}{}
    n.Predecessor = addr
  }
  <-finished
  return nil
}

func hashString(elt string) *big.Int {
    hasher := sha1.New()
    hasher.Write([]byte(elt))
    return new(big.Int).SetBytes(hasher.Sum(nil))
}

func jump(address string, fingerentry int) *big.Int {
    n := hashString(address)
    fingerentryminus1 := big.NewInt(int64(fingerentry) - 1)
    jump := new(big.Int).Exp(two, fingerentryminus1, nil)
    sum := new(big.Int).Add(n, jump)

    return new(big.Int).Mod(sum, hashMod)
}
