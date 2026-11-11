package blocklist

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

type TrancoStore struct {
	mu    sync.RWMutex
	ranks map[string]int
}

func NewTrancoStore() *TrancoStore {
	return &TrancoStore{
		ranks: make(map[string]int),
	}
}

func (s *TrancoStore) Set(domain string, rank int) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ranks[domain] = rank
}

func (s *TrancoStore) GetRank(domain string) int {
	domain = normalizeDomain(domain)
	if domain == "" {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ranks[domain]
}

func (s *TrancoStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.ranks)
}

func (s *TrancoStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ranks = make(map[string]int)
}

func (s *TrancoStore) ParseCSV(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue
		}

		rank, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		s.Set(parts[1], rank)
	}

	return scanner.Err()
}

func (s *TrancoStore) ParseAuto(r io.Reader) error {
	const zipMagicLen = 4
	var head [zipMagicLen]byte
	n, err := io.ReadFull(r, head[:])
	if err == io.EOF || (err == io.ErrUnexpectedEOF && n == 0) {
		return nil
	}
	if err != nil && err != io.ErrUnexpectedEOF {
		return fmt.Errorf("peek input: %w", err)
	}

	isZip := n >= 4 && head[0] == 'P' && head[1] == 'K' && head[2] == 0x03 && head[3] == 0x04

	if isZip {
		rest, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("read zip body: %w", err)
		}
		full := make([]byte, n+len(rest))
		copy(full, head[:n])
		copy(full[n:], rest)
		return s.ParseZip(bytes.NewReader(full), int64(len(full)))
	}

	combined := io.MultiReader(bytes.NewReader(head[:n]), r)
	return s.ParseCSV(combined)
}

func (s *TrancoStore) ParseZip(r io.ReaderAt, size int64) error {
	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}

	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, ".csv") {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("opening CSV in zip: %w", err)
			}
			defer rc.Close()

			return s.ParseCSV(rc)
		}
	}

	return fmt.Errorf("no CSV file found in zip")
}
