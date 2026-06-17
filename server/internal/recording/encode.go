// Package recording：blob 编解码 + checksum。
package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/iannil/marketing-monitor/internal/proto"
)

// encodeBlob 把一批 stream entry 编码为 MinIO 对象内容（msgpack array of envelope bytes）。
// 同时返回：blob 起止时间（最早/最晚事件 ts）、sha256 checksum。
func encodeBlob(entries []StreamEntry) (data []byte, startedAt, endedAt time.Time, checksum string, err error) {
	if len(entries) == 0 {
		return nil, time.Time{}, time.Time{}, "", errors.New("empty entries")
	}

	// 解析每个 envelope 取 ts，确定起止
	payloads := make([][]byte, 0, len(entries))
	var minTS, maxTS int64
	first := true
	for _, e := range entries {
		payloads = append(payloads, e.Data)
		env, decErr := proto.Decode(e.Data)
		if decErr == nil {
			if first {
				minTS = env.TS
				maxTS = env.TS
				first = false
			}
			if env.TS < minTS {
				minTS = env.TS
			}
			if env.TS > maxTS {
				maxTS = env.TS
			}
		}
	}
	startedAt = time.UnixMilli(minTS)
	endedAt = time.UnixMilli(maxTS)

	// 编码为 msgpack array
	data, err = msgpack.Marshal(payloads)
	if err != nil {
		return nil, time.Time{}, time.Time{}, "", err
	}

	sum := sha256.Sum256(data)
	checksum = hex.EncodeToString(sum[:])

	return data, startedAt, endedAt, checksum, nil
}
