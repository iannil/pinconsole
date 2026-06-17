// Package recording：minio RemoveObject opts（避免每处 import minio-go）。
package recording

import "github.com/minio/minio-go/v7"

var minioRemoveObjectOpts = minio.RemoveObjectOptions{}
