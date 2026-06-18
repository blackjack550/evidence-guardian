package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type Index struct {
	db *sql.DB
}

func NewIndex(dbPath string) (*Index, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开索引数据库失败: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS evidence (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		trigger_source TEXT NOT NULL,
		keyword TEXT NOT NULL,
		window_title TEXT,
		window_class TEXT,
		process_id INTEGER,
		screenshot_path TEXT,
		video_path TEXT,
		metadata_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON evidence(timestamp);
	CREATE INDEX IF NOT EXISTS idx_keyword ON evidence(keyword);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化索引数据库失败: %w", err)
	}

	return &Index{db: db}, nil
}

func (idx *Index) Insert(r *EvidenceRecord) (int64, error) {
	result, err := idx.db.Exec(
		`INSERT INTO evidence (timestamp, trigger_source, keyword, window_title, window_class, process_id, screenshot_path, video_path, metadata_path)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Timestamp.Unix(), r.TriggerSource, r.Keyword, r.WindowTitle,
		r.WindowClass, r.ProcessID, r.ScreenshotPath, r.VideoPath, r.MetadataPath,
	)
	if err != nil {
		return 0, fmt.Errorf("插入索引记录失败: %w", err)
	}
	return result.LastInsertId()
}

func (idx *Index) QueryByDate(start, end time.Time) ([]EvidenceRecord, error) {
	// TODO: implement query
	return nil, nil
}

func (idx *Index) Close() error {
	return idx.db.Close()
}
