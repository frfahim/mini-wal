package proto

import (
	"log"

	pb "google.golang.org/protobuf/proto"
)

func MarshalData(data *WAL_DATA) []byte {
	// Marshal the WALData struct to a byte slice
	marshaledData, err := pb.Marshal(data)
	if err != nil {
		log.Panicf("Error marshaling data: %v", err)
	}
	return marshaledData
}
