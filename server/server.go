package server

import (
	"time"
	"log"
	"os"
	"os/signal"
	"context"
	"sync"
	"cryptopepe.io/cryptopepe-worker/creators"
	"cryptopepe.io/cryptopepe-worker/abi/sale"
	"cryptopepe.io/cryptopepe-worker/abi/cozy"
	"cryptopepe.io/cryptopepe-worker/abi/token"
	"math/big"
	"cryptopepe.io/cryptopepe-worker/pepe"
	"cryptopepe.io/cryptopepe-svg/builder/look"
	"fmt"
)


type ContractSessions struct {
	PepeCallSession *token.TokenCallerSession
	SaleAuctionCallSession *sale.SaleCallerSession
	CozyAuctionCallSession *cozy.CozyCallerSession
}

type ImageHandlerProps struct {

	ImageCreator *creators.PepeImageCreator

	// Map of all images being successfully processed or not
	imageBuilds map[uint64]bool

	// Only build one image at a time
	imageMutex sync.Mutex
}

type Server struct {

	ContractSessions
	ImageHandlerProps
}

func (srv *Server) Start() {

	// TODO: we can pre-initialize with trues to avoid unnecessary backfilling on start-up
	srv.imageBuilds = make(map[uint64]bool)
	
	srv.imageMutex = sync.Mutex{}

	stopCh := make(chan string)
	checkPepeImages := time.NewTicker(10 * time.Second)
	// Run our task in a goroutine so that it doesn't block.
	go func() {
		for {
			select {
			case <-stopCh:
				fmt.Println("Stopping main service")
				return
			case <-checkPepeImages.C:
				srv.checkPepeImages()
			}
		}
	}()

	log.Println("Started worker!")

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 15)
	defer cancel()
	// Doesn't block if no work, but will otherwise wait
	// until the timeout deadline.
	stopCh <- "stop!"
	<- ctx.Done()

	log.Println("shutting down")
	os.Exit(0)

}

func (srv *Server) checkPepeImages() error {

	fmt.Println("Checking pepe images")

	defer srv.imageMutex.Unlock()
	srv.imageMutex.Lock()

	pepeCount, err := srv.ContractSessions.PepeCallSession.TotalSupply()
	if err != nil {
		return nil
	}
	count := pepeCount.Uint64()
	fmt.Printf("Processing pepes for image building, total count: %d\n", count)

	// ignore pepe 0, start from 1
	errCount := 0
	for	pepeId := uint64(1); pepeId < count; pepeId++ {
		if errCount > 5 {
			fmt.Println("Too many errors, something is wrong, stopping update")
		}
		if !srv.imageBuilds[pepeId] {
			fmt.Printf("Building images for pepe %d\n", pepeId)

			parsedPepe, err := srv.getPepe(big.NewInt(int64(pepeId)))
			if err != nil {
				fmt.Println(err)
				errCount++
				continue
			}
			dna := pepe.PepeDNA(parsedPepe.Genotype)
			parsedLook := (&dna).ParsePepeDNA()

			fmt.Printf("Succesfully retrieved and parsed data for pepe %d\n", pepeId)

			if err := srv.handleImage(pepeId, parsedPepe, parsedLook); err != nil {
				fmt.Println(err)
				errCount++
				continue
			}

			// Set it to true, do not rebuild next iteration.
			srv.imageBuilds[pepeId] = true
			fmt.Printf("Succesfully created images and pushed them to GC storage for pepe %d\n", pepeId)
		}
	}

	return nil
}

func (srv *Server) getPepe(pepeId *big.Int) (*pepe.Pepe, error) {

	rawPepe, err := srv.ContractSessions.PepeCallSession.GetPepe(pepeId)
	if err != nil {
		return nil, err
	}

	parsedPepe := pepe.Pepe(rawPepe)

	return &parsedPepe, nil
}

func (srv *Server) handleImage(pepeId uint64, parsedPepe *pepe.Pepe, parsedLook *look.PepeLook) error {

	err := srv.ImageCreator.Create(pepeId, parsedPepe, parsedLook, false)
	if err != nil {
		return err
	}

	return nil
}