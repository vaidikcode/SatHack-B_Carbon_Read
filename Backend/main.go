package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/rs/cors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Block EnvIndex returns carbon emission till the position of blockchain is created
type Block struct {
	Pos       int
	Data      purchaseList
	Timestamp string
	Hash      string
	PrevHash  string
	EnvIndex  string
}

type purchaseList struct {
	BookID       string `json:"book_id"`
	User         string `json:"user"`
	CheckoutDate string `json:"checkout_date"`
	IsGenesis    bool   `json:"is_genesis"`
}

type Book struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	PublishDate string `json:"publish_date"`
	ISBN        string `json:"isbn"`
}

var carbonEmissionOfAllBlocks float64

func (b *Block) generateHash() {
	bytes, _ := json.Marshal(b.Data)
	data := string(b.Pos) + b.Timestamp + string(bytes) + b.PrevHash
	hash := sha256.New()
	hash.Write([]byte(data))
	b.Hash = hex.EncodeToString(hash.Sum(nil))
}

func CreateBlock(prevBlock *Block, checkoutItem purchaseList) *Block {
	block := &Block{}
	block.Pos = prevBlock.Pos + 1
	block.Timestamp = time.Now().String()
	block.Data = checkoutItem
	block.PrevHash = prevBlock.Hash
	block.generateHash()

	return block
}

type Blockchain struct {
	blocks []*Block
}

var BlockChain *Blockchain

func (bc *Blockchain) AddBlock(data purchaseList) {

	prevBlock := bc.blocks[len(bc.blocks)-1]

	block := CreateBlock(prevBlock, data)

	if validBlock(block, prevBlock) {
		bc.blocks = append(bc.blocks, block)
	}
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, purchaseList{IsGenesis: true})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func validBlock(block, prevBlock *Block) bool {

	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if !block.validateHash(block.Hash) {
		return false
	}

	if prevBlock.Pos+1 != block.Pos {
		return false
	}
	return true
}

func (b *Block) validateHash(hash string) bool {
	b.generateHash()
	if b.Hash != hash {
		return false
	}
	return true
}

func getBlockchain(w http.ResponseWriter, r *http.Request) {
	jbytes, err := json.MarshalIndent(BlockChain.blocks, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	io.WriteString(w, string(jbytes))
}

func writeBlock(w http.ResponseWriter, r *http.Request) {
	var checkoutItem purchaseList
	if err := json.NewDecoder(r.Body).Decode(&checkoutItem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not write Block: %v", err)
		w.Write([]byte("could not write block"))
		return
	}

	BlockChain.AddBlock(checkoutItem)
	resp, err := json.MarshalIndent(checkoutItem, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not write block"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func newBook(w http.ResponseWriter, r *http.Request) {
	var book Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not create: %v", err)
		w.Write([]byte("could not create new Book"))
		return
	}

	h := md5.New()
	io.WriteString(h, book.ISBN+book.PublishDate)
	book.ID = fmt.Sprintf("%x", h.Sum(nil))

	resp, err := json.MarshalIndent(book, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not save book data"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func CarbonEmissionofSingleBlock() float64 {
	ef := float64(480)
	value := (ef * 0.15) + (0.01 * 750 * 10)
	return value
}

func calculateEmissionRate(w http.ResponseWriter, r *http.Request) {
	//We are calculating here all the carbon emission of the blockchain till how many blocks are generated
	bp := float64(len(BlockChain.blocks))
	carbonEmissionOfAllBlocks = CarbonEmissionofSingleBlock() * bp
	response := map[string]string{
		"carbonEmission": fmt.Sprintf("%.2f", carbonEmissionOfAllBlocks),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getLeaderBoard(w http.ResponseWriter, r *http.Request) {
	bp := float64(len(BlockChain.blocks))
	carbonEmissionOfAllBlocks = CarbonEmissionofSingleBlock() * bp
	response := map[string]string{
		"carbonEmission": fmt.Sprintf("%.2f", carbonEmissionOfAllBlocks),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {

	BlockChain = NewBlockchain()

	r := mux.NewRouter()

	r.HandleFunc("/blockdisplay", getBlockchain).Methods("GET")
	r.HandleFunc("/blockdisplay", writeBlock).Methods("POST")
	r.HandleFunc("/checkout", newBook).Methods("POST")
	r.HandleFunc("/emissionrate", calculateEmissionRate).Methods("GET")
	r.HandleFunc("/leaderboard", getLeaderBoard).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           86400,
	})
	handler := c.Handler(r)
	go func() {

		for _, block := range BlockChain.blocks {
			fmt.Printf("Prev. hash: %x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data: %v\n", string(bytes))
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Printf("%x\n", carbonEmissionOfAllBlocks)
			fmt.Println()
		}

	}()
	log.Println("Listening on port 3000")

	log.Fatal(http.ListenAndServe(":3000", handler))
}
