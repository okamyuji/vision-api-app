-- Database initialization script
SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

CREATE DATABASE IF NOT EXISTS household CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE household;

-- Receipts table
CREATE TABLE IF NOT EXISTS receipts (
    id VARCHAR(36) PRIMARY KEY,
    store_name VARCHAR(255) NOT NULL,
    purchase_date DATETIME NOT NULL,
    total_amount INT NOT NULL COMMENT '実際に使った金額',
    tax_amount INT NOT NULL DEFAULT 0 COMMENT '消費税額',
    payment_method VARCHAR(50) DEFAULT '' COMMENT '支払い方法',
    receipt_number VARCHAR(100) DEFAULT '' COMMENT 'レシート番号',
    category VARCHAR(50),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_purchase_date (purchase_date),
    INDEX idx_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Receipt items table
CREATE TABLE IF NOT EXISTS receipt_items (
    id VARCHAR(36) PRIMARY KEY,
    receipt_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    price INT NOT NULL,
    category VARCHAR(50) COMMENT '明細項目のカテゴリー',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (receipt_id) REFERENCES receipts(id) ON DELETE CASCADE,
    INDEX idx_receipt_id (receipt_id),
    INDEX idx_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Expense entries table
CREATE TABLE IF NOT EXISTS expense_entries (
    id VARCHAR(36) PRIMARY KEY,
    receipt_id VARCHAR(36),
    date DATETIME NOT NULL,
    category VARCHAR(50) NOT NULL,
    amount INT NOT NULL,
    description TEXT,
    tags JSON,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (receipt_id) REFERENCES receipts(id) ON DELETE SET NULL,
    INDEX idx_date (date),
    INDEX idx_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Categories table
CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    color VARCHAR(7),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default categories
INSERT INTO categories (id, name, description, color) VALUES
    (UUID(), '食費', '食料品・飲料', '#FF6B6B'),
    (UUID(), '日用品', '生活用品・雑貨', '#4ECDC4'),
    (UUID(), '交通費', '電車・バス・タクシー', '#45B7D1'),
    (UUID(), '医療費', '病院・薬局', '#96CEB4'),
    (UUID(), '娯楽費', 'レジャー・趣味', '#FFEAA7'),
    (UUID(), '通信費', '携帯・インターネット', '#DFE6E9'),
    (UUID(), '光熱費', '電気・ガス・水道', '#74B9FF'),
    (UUID(), 'その他', 'その他の支出', '#A29BFE')
ON DUPLICATE KEY UPDATE name=name;

