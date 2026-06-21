// Package storage adalah klien penyimpanan objek S3-compatible (idcloudhost).
// Pure-Go (minio-go), aman untuk build CGO_ENABLED=0 + distroless. Objek di-upload
// public-read sehingga bisa disajikan via URL langsung yang cache-friendly.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/elkasir/api/internal/platform/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client membungkus minio-go untuk satu bucket + prefix dasar.
type Client struct {
	mc         *minio.Client
	bucket     string
	basePath   string
	publicBase string
}

// New membangun klien storage dari konfigurasi. Tidak melakukan koneksi jaringan
// (minio-go bersifat lazy) — error hanya muncul bila endpoint/kredensial salah bentuk.
func New(cfg config.ObjectStorage) (*Client, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1" // default penandatanganan v4 untuk provider S3-compatible
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       region,
		BucketLookup: minio.BucketLookupPath, // idcloudhost memakai path-style URL
	})
	if err != nil {
		return nil, err
	}

	publicBase := strings.TrimRight(cfg.PublicBaseURL, "/")
	if publicBase == "" {
		scheme := "https"
		if !cfg.UseSSL {
			scheme = "http"
		}
		publicBase = fmt.Sprintf("%s://%s/%s", scheme, cfg.Endpoint, cfg.Bucket)
	}

	return &Client{
		mc:         mc,
		bucket:     cfg.Bucket,
		basePath:   strings.Trim(cfg.BasePath, "/"),
		publicBase: publicBase,
	}, nil
}

// Put menyimpan data di <basePath>/<category>/<name> dengan ACL public-read dan
// cache panjang (objek immutable karena nama ber-ULID), lalu mengembalikan key &
// URL publiknya.
func (c *Client) Put(ctx context.Context, category, name, contentType string, data []byte) (key, url string, err error) {
	key = path.Join(c.basePath, category, name)
	_, err = c.mc.PutObject(ctx, c.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType:    contentType,
		CacheControl:   "public, max-age=31536000, immutable",
		SendContentMd5: true,
		// Canned ACL via header langsung (minio-go mengenali "x-amz-acl").
		UserMetadata: map[string]string{"x-amz-acl": "public-read"},
	})
	if err != nil {
		return "", "", err
	}
	return key, c.publicBase + "/" + key, nil
}
