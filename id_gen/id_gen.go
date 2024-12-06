package id_gen

import (
	"crypto/rand"
	"encoding/hex"
	mrand "math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// region interface
// GenerateUUID generates a new UUID
func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateUuidWithPrefix(prefix string) string {
	return prefix + "-" + GenerateUUID()
}

// GenerateSnowflakeID generates a new Snowflake ID using the singleton generator
func GenerateSnowflakeID() int64 {
	once.Do(initSnowflakeGenerator)
	return snowflakeGenerator.GenerateSnowflakeID()
}

func GenerateRandomHexString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

func GenerateSortableId() string {
	entropy := ulid.Monotonic(mrand.New(mrand.NewSource(time.Now().UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// endregion

// region Snowflake id generator details

var (
	snowflakeGenerator *SnowflakeGenerator
	once               sync.Once
)

// initSnowflakeGenerator initializes the singleton SnowflakeGenerator
func initSnowflakeGenerator() {
	machineID := getMachineID()
	snowflakeGenerator = NewSnowflakeGenerator(machineID)
}

// getMachineID attempts to get a unique machine ID
func getMachineID() int64 {
	// Try to get the last part of the IP address
	if ip, err := getLastIPOctet(); err == nil {
		return int64(ip)
	}

	// Fallback to using the process ID
	pid := os.Getpid()
	return int64(pid % 1024)
}

// getLastIPOctet gets the last octet of the first non-loopback IP address
func getLastIPOctet() (int, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return 0, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				return int(ipv4[3]), nil
			}
		}
	}
	return 0, nil
}

// SnowflakeGenerator is a struct to generate Snowflake IDs
type SnowflakeGenerator struct {
	mutex         sync.Mutex
	lastTimestamp int64
	sequence      int64
	machineID     int64
}

// NewSnowflakeGenerator creates a new SnowflakeGenerator
func NewSnowflakeGenerator(machineID int64) *SnowflakeGenerator {
	return &SnowflakeGenerator{
		lastTimestamp: 0,
		sequence:      0,
		machineID:     machineID & 0x3FF, // Ensure machineID is 10 bits
	}
}

// GenerateSnowflakeID generates a new Snowflake ID
func (sg *SnowflakeGenerator) GenerateSnowflakeID() int64 {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	timestamp := time.Now().UnixMilli()

	if timestamp == sg.lastTimestamp {
		sg.sequence = (sg.sequence + 1) & 0xFFF
		if sg.sequence == 0 {
			timeout := time.After(time.Millisecond)
			for timestamp <= sg.lastTimestamp {
				select {
				case <-timeout:
					// If we've waited too long, generate a new timestamp
					timestamp = sg.lastTimestamp + 1
				default:
					timestamp = time.Now().UnixMilli()
				}
			}
		}
	} else {
		sg.sequence = 0
	}

	sg.lastTimestamp = timestamp

	return (timestamp << 22) | (sg.machineID << 12) | sg.sequence
}

// endregion
