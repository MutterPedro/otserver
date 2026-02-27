// Package protocol defines shared constants and types for the Tibia network
// protocol, used across the buffer, crypto, and network packages.
package protocol

// MaxNetworkMessageSize is the maximum size in bytes of a single Tibia network
// message. Both the buffer and checksum subsystems enforce this limit.
const MaxNetworkMessageSize = 24590
