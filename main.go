package main

//imports
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

//block of blockchain
type Block struct {
	Index int
	Timestamp string
	Temperature int
	Humidity int
	ProductId string
	Hash string
	PrevHash string
	FarmId string
	ProductQuality string
}

//blockchain
var Blockchain []Block

//index number
var end = 1

//current block hash calculation
func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.Temperature) + string(block.Humidity) + block.ProductId + block.PrevHash + block.FarmId + block.ProductQuality
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

//appending to blockchain
func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

//error
func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

//request handling
func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/GetDetails", handleWriteBlockForCustomer).Methods("POST")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

//write response
func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

//farm request
type Message struct {
	Temperature int
	Humidity int
	ProductId string
	FarmId string
}

//product accepted/rejected
type Rejected struct {
	RejectedMsg string
}

//write block
func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
	var m Message
	var rej Rejected

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	var newBlock Block

	t := time.Now()

	newBlock.Index = end
	newBlock.Timestamp = t.String()
	newBlock.Temperature = m.Temperature
	newBlock.Humidity = m.Humidity
	newBlock.ProductId = m.ProductId
	newBlock.PrevHash = Blockchain[len(Blockchain)-1].Hash
	newBlock.Hash = calculateHash(newBlock)
	newBlock.FarmId = m.FarmId

	//product quality determined
	if(m.Temperature == 25 && m.Humidity == 45) {
		newBlock.ProductQuality = "Excellent"
	} else if(m.Temperature < 25 && m.Humidity < 45) {
		newBlock.ProductQuality = "Very Good"
	} else if(m.Temperature > 25 && m.Humidity > 45) {
		newBlock.ProductQuality = "Good"
	} else {
		newBlock.ProductQuality = "Average"
	}

	//product rejected
	if(newBlock.Index != 0)	{
		if(m.Temperature < 20 || m.Temperature > 30 || m.Humidity > 50 || m.Humidity < 40) {
			rej.RejectedMsg = "Product not accepted. Assessment test failed."
			respondWithJSON(w, r, 406, rej)
		} else {
			respondWithJSON(w, r, http.StatusCreated, newBlock)
		}
	}
	
	//validation product accepted
	if(newBlock.Temperature >= 20 && newBlock.Temperature <= 30 && newBlock.Humidity >= 40 && newBlock.Humidity <= 50 && newBlock.FarmId != "") {
		end = end + 1
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}
}

//customer request
type pId struct {
	ProductId string
}

//customer response
type Product struct {
	Temperature int
	Humidity int
	ProductId string
	ProductQuality string
	FarmId string
	IsFound string
}

//write response to customer
func handleWriteBlockForCustomer(w http.ResponseWriter, r *http.Request) {
	var m pId

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	var flag int
	flag = 0

	var pIdBlock Product
	for i := 0; i < len(Blockchain); i++ {
		if(Blockchain[i].ProductId == m.ProductId) {
			pIdBlock.Temperature = Blockchain[i].Temperature
			pIdBlock.Humidity = Blockchain[i].Humidity
			pIdBlock.ProductId = Blockchain[i].ProductId
			pIdBlock.ProductQuality = Blockchain[i].ProductQuality
			pIdBlock.FarmId = Blockchain[i].FarmId
			pIdBlock.IsFound = "Product details retrieved."
			flag = 1
			respondWithJSON(w, r, 200, pIdBlock)
		}
	}

	if(flag == 0) {
		pIdBlock.Temperature = 0
		pIdBlock.Humidity = 0
		pIdBlock.ProductId = m.ProductId
		pIdBlock.ProductQuality = "NA"
		pIdBlock.FarmId = "NA"
		pIdBlock.IsFound = "Product not found."
		flag = 0
		respondWithJSON(w, r, 404, pIdBlock)
	}
	
}

//json response
func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

//main with genesis block
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{0, t.String(), 0, 0, "", "dc53079703d684e6f7c4c08a32d9cf878b9d7ee0fd7bac3e73e97ffb21d15f34", "", "", ""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()
	log.Fatal(run())

}



//execute details
//go run main.go 8000
//gedit .env ADDR=8000

//http://localhost:8000/
//{"Temperature": 24,
//"Humidity": 45,
//"ProductId": "abc",
//"FarmId":"farm1"}

//http://localhost:8000/GetDetails
//{"ProductId": "p1"}


