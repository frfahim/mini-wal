package proto

import (
	"log"

	"wal/proto/wal_pb"

	"google.golang.org/protobuf/proto"
)

func MarshalData(data *wal_pb.WAL_DATA) []byte {
	// Marshal the WALData struct to a byte slice
	marshaledData, err := proto.Marshal(data)
	if err != nil {
		log.Panicf("Error marshaling data: %v", err)
	}
	return marshaledData
}
