package main

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"flag"
	"fmt"
	"cryptopepe.io/worker/abi/token"
	"cryptopepe.io/worker/abi/sale"
	"cryptopepe.io/worker/abi/cozy"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"cryptopepe.io/worker/server"
	"cryptopepe.io/worker/creators"
	"context"
	"cloud.google.com/go/storage"
)

func main() {

	rpcAddress := flag.String("rpc",
		"http://localhost:8545",
		"http/ws/ipc RPC connection to full node.")
	tokenAddress := flag.String("token-address",
		"0x1234abcd1234abcd1234abcd1234abcd1234abcd",
		"Pepe token address.")
	saleAuctionAddress := flag.String("sale-auction-address",
		"0x1234abcd1234abcd1234abcd1234abcd1234abcd",
		"Pepe sale auction token address.")
	cozyAuctionAddress := flag.String("cozy-auction-address",
		"0x1234abcd1234abcd1234abcd1234abcd1234abcd",
		"Cozy time auction token address.")

	//parse flags!
	flag.Parse()

	fmt.Printf(`Started!
		truffle net id: %v
		rpc: %s
		token address: %s
		saleAuctionAddress: %s
		cozyAuctionAddress: %s
`, *rpcAddress, *tokenAddress, *saleAuctionAddress, *cozyAuctionAddress)

	var rawPepeToken *token.Token
	var rawSaleAuctionToken *sale.Sale
	var rawCozyAuctionToken *cozy.Cozy

	ethClient := NewEthConnection(*rpcAddress)

	var err error
	// Instantiate the contract
	rawPepeToken, err = token.NewToken(common.HexToAddress(*tokenAddress), ethClient)
	if err != nil {
		log.Fatalf("Failed to instantiate a Token contract: %v", err)
	}
	rawSaleAuctionToken, err = sale.NewSale(common.HexToAddress(*saleAuctionAddress), ethClient)
	if err != nil {
		log.Fatalf("Failed to instantiate a Token contract: %v", err)
	}
	rawCozyAuctionToken, err = cozy.NewCozy(common.HexToAddress(*cozyAuctionAddress), ethClient)
	if err != nil {
		log.Fatalf("Failed to instantiate a Token contract: %v", err)
	}

	// Creates a client.
	client, err := storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Sets the name for the new bucket.
	bucketName := "pepes"

	// Creates a Bucket instance.
	pepeImgBucket := client.Bucket(bucketName)

	srv := new(server.Server)
	srv.ImageCreator = creators.NewPepeImageCreator(pepeImgBucket)


	// Wrap the Token contract instance into a caller-session (session without auth)
	srv.ContractSessions.PepeCallSession = &token.TokenCallerSession{
		Contract: &rawPepeToken.TokenCaller,
		CallOpts: bind.CallOpts{
			Pending: false,//ignore pending data.
		},
	}

	// Wrap the Token contract instance into a caller-session (session without auth)
	srv.SaleAuctionCallSession = &sale.SaleCallerSession{
		Contract: &rawSaleAuctionToken.SaleCaller,
		CallOpts: bind.CallOpts{
			Pending: false,//ignore pending data.
		},
	}

	// Wrap the Token contract instance into a caller-session (session without auth)
	srv.CozyAuctionCallSession = &cozy.CozyCallerSession{
		Contract: &rawCozyAuctionToken.CozyCaller,
		CallOpts: bind.CallOpts{
			Pending: false,//ignore pending data.
		},
	}

	// Server will handle shutdown
	srv.Start()
}

// Connect to a full node
func NewEthConnection(rpcAddress string) *ethclient.Client {

	c, err := rpc.Dial(rpcAddress)
	if err != nil {
		log.Fatalln("Could not connect to RPC!", err)
	}

	// Create a RPC connection to a full node
	conn := ethclient.NewClient(c)

	fmt.Println("Dial ok!")

	return conn
}
