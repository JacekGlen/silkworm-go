//go:build !nosilkworm && unix && !(linux && arm64)

package silkworm_go

// #cgo CFLAGS: -I${SRCDIR}/include
// #include "silkworm.h"
import "C"

/*

#include <stdlib.h>
#include <string.h>

static bool go_string_copy(_GoString_ s, char *dest, size_t size) {
	size_t len = _GoStringLen(s);
	if (len >= size) return false;
	const char *src = _GoStringPtr(s);
	strncpy(dest, src, len);
	dest[len] = '\0';
	return true;
}

*/
import "C"

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"unsafe"
)

const (
	SILKWORM_OK                      = C.SILKWORM_OK
	SILKWORM_INTERNAL_ERROR          = C.SILKWORM_INTERNAL_ERROR
	SILKWORM_UNKNOWN_ERROR           = C.SILKWORM_UNKNOWN_ERROR
	SILKWORM_INVALID_HANDLE          = C.SILKWORM_INVALID_HANDLE
	SILKWORM_INVALID_PATH            = C.SILKWORM_INVALID_PATH
	SILKWORM_INVALID_SNAPSHOT        = C.SILKWORM_INVALID_SNAPSHOT
	SILKWORM_INVALID_MDBX_TXN        = C.SILKWORM_INVALID_MDBX_TXN
	SILKWORM_INVALID_BLOCK_RANGE     = C.SILKWORM_INVALID_BLOCK_RANGE
	SILKWORM_BLOCK_NOT_FOUND         = C.SILKWORM_BLOCK_NOT_FOUND
	SILKWORM_UNKNOWN_CHAIN_ID        = C.SILKWORM_UNKNOWN_CHAIN_ID
	SILKWORM_MDBX_ERROR              = C.SILKWORM_MDBX_ERROR
	SILKWORM_INVALID_BLOCK           = C.SILKWORM_INVALID_BLOCK
	SILKWORM_DECODING_ERROR          = C.SILKWORM_DECODING_ERROR
	SILKWORM_TOO_MANY_INSTANCES      = C.SILKWORM_TOO_MANY_INSTANCES
	SILKWORM_INVALID_SETTINGS        = C.SILKWORM_INVALID_SETTINGS
	SILKWORM_TERMINATION_SIGNAL      = C.SILKWORM_TERMINATION_SIGNAL
	SILKWORM_SERVICE_ALREADY_STARTED = C.SILKWORM_SERVICE_ALREADY_STARTED
	SILKWORM_INCOMPATIBLE_LIBMDBX    = C.SILKWORM_INCOMPATIBLE_LIBMDBX
)

// ErrInterrupted is the error returned by Silkworm APIs when stopped by any termination signal.
var ErrInterrupted = errors.New("interrupted")
var ErrInvalidBlock = errors.New("invalid block")

type Silkworm struct {
	handle C.SilkwormHandle
}

func New(dataDirPath string, libMdbxVersion string) (*Silkworm, error) {
	silkworm := &Silkworm{
		handle: nil,
	}

	settings := &C.struct_SilkwormSettings{}

	if !C.go_string_copy(dataDirPath, &settings.data_dir_path[0], C.SILKWORM_PATH_SIZE) {
		return nil, errors.New("silkworm.New failed to copy dataDirPath")
	}

	if !C.go_string_copy(libMdbxVersion, &settings.libmdbx_version[0], 32) {
		return nil, errors.New("silkworm.New failed to copy libMdbxVersion")
	}

	status := C.silkworm_init(&silkworm.handle, settings) //nolint:gocritic
	if status == SILKWORM_OK {
		return silkworm, nil
	}
	return nil, fmt.Errorf("silkworm_init error %d", status)
}

func (s *Silkworm) Close() error {
	status := C.silkworm_fini(s.handle)
	s.handle = nil
	if status == SILKWORM_OK {
		return nil
	}
	return fmt.Errorf("silkworm_fini error %d", status)
}

func (s *Silkworm) AddSnapshot(snapshot *MappedChainSnapshot) error {
	cHeadersSegmentFilePath := C.CString(snapshot.Headers.Segment.FilePath)
	defer C.free(unsafe.Pointer(cHeadersSegmentFilePath))
	cHeadersIdxHeaderHashFilePath := C.CString(snapshot.Headers.IdxHeaderHash.FilePath)
	defer C.free(unsafe.Pointer(cHeadersIdxHeaderHashFilePath))
	cHeadersSnapshot := C.struct_SilkwormHeadersSnapshot{
		segment: C.struct_SilkwormMemoryMappedFile{
			file_path:      cHeadersSegmentFilePath,
			memory_address: (*C.uchar)(snapshot.Headers.Segment.DataHandle),
			memory_length:  C.uint64_t(snapshot.Headers.Segment.Size),
		},
		header_hash_index: C.struct_SilkwormMemoryMappedFile{
			file_path:      cHeadersIdxHeaderHashFilePath,
			memory_address: (*C.uchar)(snapshot.Headers.IdxHeaderHash.DataHandle),
			memory_length:  C.uint64_t(snapshot.Headers.IdxHeaderHash.Size),
		},
	}

	cBodiesSegmentFilePath := C.CString(snapshot.Bodies.Segment.FilePath)
	defer C.free(unsafe.Pointer(cBodiesSegmentFilePath))
	cBodiesIdxBodyNumberFilePath := C.CString(snapshot.Bodies.IdxBodyNumber.FilePath)
	defer C.free(unsafe.Pointer(cBodiesIdxBodyNumberFilePath))
	cBodiesSnapshot := C.struct_SilkwormBodiesSnapshot{
		segment: C.struct_SilkwormMemoryMappedFile{
			file_path:      cBodiesSegmentFilePath,
			memory_address: (*C.uchar)(snapshot.Bodies.Segment.DataHandle),
			memory_length:  C.uint64_t(snapshot.Bodies.Segment.Size),
		},
		block_num_index: C.struct_SilkwormMemoryMappedFile{
			file_path:      cBodiesIdxBodyNumberFilePath,
			memory_address: (*C.uchar)(snapshot.Bodies.IdxBodyNumber.DataHandle),
			memory_length:  C.uint64_t(snapshot.Bodies.IdxBodyNumber.Size),
		},
	}

	cTxsSegmentFilePath := C.CString(snapshot.Txs.Segment.FilePath)
	defer C.free(unsafe.Pointer(cTxsSegmentFilePath))
	cTxsIdxTxnHashFilePath := C.CString(snapshot.Txs.IdxTxnHash.FilePath)
	defer C.free(unsafe.Pointer(cTxsIdxTxnHashFilePath))
	cTxsIdxTxnHash2BlockFilePath := C.CString(snapshot.Txs.IdxTxnHash2BlockNum.FilePath)
	defer C.free(unsafe.Pointer(cTxsIdxTxnHash2BlockFilePath))
	cTxsSnapshot := C.struct_SilkwormTransactionsSnapshot{
		segment: C.struct_SilkwormMemoryMappedFile{
			file_path:      cTxsSegmentFilePath,
			memory_address: (*C.uchar)(snapshot.Txs.Segment.DataHandle),
			memory_length:  C.uint64_t(snapshot.Txs.Segment.Size),
		},
		tx_hash_index: C.struct_SilkwormMemoryMappedFile{
			file_path:      cTxsIdxTxnHashFilePath,
			memory_address: (*C.uchar)(snapshot.Txs.IdxTxnHash.DataHandle),
			memory_length:  C.uint64_t(snapshot.Txs.IdxTxnHash.Size),
		},
		tx_hash_2_block_index: C.struct_SilkwormMemoryMappedFile{
			file_path:      cTxsIdxTxnHash2BlockFilePath,
			memory_address: (*C.uchar)(snapshot.Txs.IdxTxnHash2BlockNum.DataHandle),
			memory_length:  C.uint64_t(snapshot.Txs.IdxTxnHash2BlockNum.Size),
		},
	}

	cChainSnapshot := C.struct_SilkwormChainSnapshot{
		headers:      cHeadersSnapshot,
		bodies:       cBodiesSnapshot,
		transactions: cTxsSnapshot,
	}

	status := C.silkworm_add_snapshot(s.handle, &cChainSnapshot) //nolint:gocritic
	if status == SILKWORM_OK {
		return nil
	}
	return fmt.Errorf("silkworm_add_snapshot error %d", status)
}

func (s *Silkworm) LibMdbxVersion() string {
	return C.GoString(C.silkworm_libmdbx_version())
}

func (s *Silkworm) StartRpcDaemon(dbEnvCHandle unsafe.Pointer) error {
	cEnv := (*C.MDBX_env)(dbEnvCHandle)
	status := C.silkworm_start_rpcdaemon(s.handle, cEnv)
	// Handle successful execution
	if status == SILKWORM_OK {
		return nil
	}
	return fmt.Errorf("silkworm_start_rpcdaemon error %d", status)
}

func (s *Silkworm) StopRpcDaemon() error {
	status := C.silkworm_stop_rpcdaemon(s.handle)
	// Handle successful execution
	if status == SILKWORM_OK {
		return nil
	}
	return fmt.Errorf("silkworm_stop_rpcdaemon error %d", status)
}

type SentrySettings struct {
	ClientId    string
	ApiPort     int
	Port        int
	Nat         string
	NetworkId   uint64
	NodeKey     []byte
	StaticPeers []string
	Bootnodes   []string
	NoDiscover  bool
	MaxPeers    int
}

func copyPeerURLs(list []string, cList *[C.SILKWORM_SENTRY_SETTINGS_PEERS_MAX][C.SILKWORM_SENTRY_SETTINGS_PEER_URL_SIZE]C.char) error {
	listLen := len(list)
	if listLen > C.SILKWORM_SENTRY_SETTINGS_PEERS_MAX {
		return errors.New("copyPeerURLs: peers URL list has too many items")
	}
	// mark the list end with an empty string
	if listLen < C.SILKWORM_SENTRY_SETTINGS_PEERS_MAX {
		cList[listLen][0] = 0
	}
	for i, url := range list {
		if !C.go_string_copy(url, &cList[i][0], C.SILKWORM_SENTRY_SETTINGS_PEER_URL_SIZE) {
			return fmt.Errorf("copyPeerURLs: failed to copy peer URL %d", i)
		}
	}
	return nil
}

func makeCSentrySettings(settings SentrySettings) (*C.struct_SilkwormSentrySettings, error) {
	cSettings := &C.struct_SilkwormSentrySettings{
		api_port:    C.uint16_t(settings.ApiPort),
		port:        C.uint16_t(settings.Port),
		network_id:  C.uint64_t(settings.NetworkId),
		no_discover: C.bool(settings.NoDiscover),
		max_peers:   C.size_t(settings.MaxPeers),
	}
	if !C.go_string_copy(settings.ClientId, &cSettings.client_id[0], C.SILKWORM_SENTRY_SETTINGS_CLIENT_ID_SIZE) {
		return nil, errors.New("makeCSentrySettings failed to copy ClientId")
	}
	if !C.go_string_copy(settings.Nat, &cSettings.nat[0], C.SILKWORM_SENTRY_SETTINGS_NAT_SIZE) {
		return nil, errors.New("makeCSentrySettings failed to copy Nat")
	}
	if len(settings.NodeKey) == C.SILKWORM_SENTRY_SETTINGS_NODE_KEY_SIZE {
		C.memcpy(unsafe.Pointer(&cSettings.node_key[0]), unsafe.Pointer(&settings.NodeKey[0]), C.SILKWORM_SENTRY_SETTINGS_NODE_KEY_SIZE) //nolint:gocritic
	} else {
		return nil, errors.New("makeCSentrySettings failed to copy NodeKey")
	}
	if err := copyPeerURLs(settings.StaticPeers, &cSettings.static_peers); err != nil {
		return nil, fmt.Errorf("copyPeerURLs failed to copy StaticPeers: %w", err)
	}
	if err := copyPeerURLs(settings.Bootnodes, &cSettings.bootnodes); err != nil {
		return nil, fmt.Errorf("copyPeerURLs failed to copy Bootnodes: %w", err)
	}
	return cSettings, nil
}

func (s *Silkworm) SentryStart(settings SentrySettings) error {
	cSettings, err := makeCSentrySettings(settings)
	if err != nil {
		return err
	}
	status := C.silkworm_sentry_start(s.handle, cSettings)
	if status == SILKWORM_OK {
		return nil
	}
	return fmt.Errorf("silkworm_sentry_start error %d", status)
}

func (s *Silkworm) SentryStop() error {
	status := C.silkworm_sentry_stop(s.handle)
	if status == SILKWORM_OK {
		return nil
	}
	return fmt.Errorf("silkworm_sentry_stop error %d", status)
}

func (s *Silkworm) ExecuteBlocks(
	txnCHandle unsafe.Pointer,
	chainID *big.Int,
	startBlock uint64,
	maxBlock uint64,
	batchSize uint64,
	writeChangeSets,
	writeReceipts,
	writeCallTraces bool,
) (lastExecutedBlock uint64, err error) {
	if runtime.GOOS == "darwin" {
		return 0, errors.New("silkworm execution is incompatible with Go runtime on macOS due to stack size mismatch (see https://github.com/golang/go/issues/28024)")
	}

	cTxn := (*C.MDBX_txn)(txnCHandle)
	cChainId := C.uint64_t(chainID.Uint64())
	cStartBlock := C.uint64_t(startBlock)
	cMaxBlock := C.uint64_t(maxBlock)
	cBatchSize := C.uint64_t(batchSize)
	cWriteChangeSets := C._Bool(writeChangeSets)
	cWriteReceipts := C._Bool(writeReceipts)
	cWriteCallTraces := C._Bool(writeCallTraces)
	cLastExecutedBlock := C.uint64_t(startBlock - 1)
	cMdbxErrorCode := C.int(0)
	status := C.silkworm_execute_blocks(
		s.handle,
		cTxn,
		cChainId,
		cStartBlock,
		cMaxBlock,
		cBatchSize,
		cWriteChangeSets,
		cWriteReceipts,
		cWriteCallTraces,
		&cLastExecutedBlock,
		&cMdbxErrorCode,
	)
	lastExecutedBlock = uint64(cLastExecutedBlock)
	// Handle successful execution
	if status == SILKWORM_OK {
		return lastExecutedBlock, nil
	}
	// Handle special errors
	if status == SILKWORM_INVALID_BLOCK {
		return lastExecutedBlock, ErrInvalidBlock
	}
	if status == SILKWORM_TERMINATION_SIGNAL {
		return lastExecutedBlock, ErrInterrupted
	}
	return lastExecutedBlock, fmt.Errorf("silkworm_execute_blocks error %d, MDBX error %d", status, cMdbxErrorCode)
}
