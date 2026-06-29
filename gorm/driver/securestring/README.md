# SecureString Driver

The `securestring` package provides a custom data type (`SecureStringDriver`) for Golang. It supports automatic encryption when saving to the database and lazy decryption when reading, utilizing a KMS provider.

This package is compatible with the `database/sql` interface and popular ORMs like **GORM**.

## How it Works

1.  **Write (Save to DB):**
    * Input: Raw text (sensitive string).
    * Process: Call KMS Encrypt -> Get `EncryptedInfo` struct -> JSON Marshal -> Base64 Encode.
    * Output: Base64 string is stored in the DB column.

2.  **Read (Load from DB):**
    * Input: Base64 string from DB.
    * Process (Scan): Base64 Decode -> JSON Unmarshal -> Store `EncryptedInfo` in the struct (Decryption does **not** happen yet).
    * Process (String): When `.String(ctx)` is called, the system triggers KMS Decrypt to retrieve the Raw text.

## Setup & Configuration

Before using the driver, you **must** inject the `KmsInterface` into the package. This is typically done in `main.go` or during the application bootstrap phase.

```go
package main

import (
    "github.com/minhgiang16983/Minh-Kit-Hehe/kms"
    "github.com/minhgiang16983/Minh-Kit-Hehe/gorm/driver/securestring"
)

func main() {
    // 1. Initialize KMS Client (e.g., AWS KMS or Mock)
    kmsClient := kms.NewKMSClient(...) 

    // 2. Inject into the securestring package
    securestring.SetKmsClient(kmsClient)

    // ... start server code
}

```

## Usage with GORM

To use `SecureStringDriver` with GORM, simply use it as the type for the fields you want to encrypt in your model struct.

### 1. Define the Model

Use `*securestring.SecureStringDriver` for encrypted fields.

**Important:** Since the encrypted data (JSON + Base64) is significantly larger than the raw text, you should define the database column type as `TEXT`, `MEDIUMTEXT`, or `LONGTEXT` (depending on your data size) instead of `VARCHAR`.

```go
import (
    "github.com/minhgiang16983/Minh-Kit-Hehe/gorm/driver/securestring"
)

type UserApiKey struct {
    ID      uint
    UserId  string
    // This field will be encrypted
    ApiKey  *securestring.SecureStringDriver `gorm:"type:text"`
}
```

### 2. Save data (NewSecureStringDriver)

To save a plain text string to the database in an encrypted format, initialize the field using `NewSecureStringDriver(rawText)`.

This function does not call KMS immediately. It prepares the struct so that the actual encryption occurs automatically when GORM executes the `Create` or `Save` operation.


```go
func CreateKey(db *gorm.DB, userID string, rawKey string) error {
    // rawKey: "sk_live_123456..." (Plain text)
    
    data := UserApiKey{
        UserId: userID,
        // Wrap the raw string into the Driver
        ApiKey: securestring.NewSecureStringDriver(rawKey),
    }

    // When this executes:
    // The Driver automatically calls KMS Encrypt -> Returns Base64 String -> Saves to DB
    return db.Create(&data).Error
}
```

### 3. Reading data

The package uses a Lazy Decryption mechanism.

1. When you query data (`db.Find`, `db.First`), the record is loaded, but the sensitive field still contains the encrypted metadata, not the plain text.

2. You must explicitly call the `.String(ctx)` method to trigger the KMS decryption and retrieve the plain text.

```go
func GetKey(ctx context.Context, db *gorm.DB, id uint) (string, error) {
    var data UserApiKey

    // Step 1: Query from DB
    // At this point, data.ApiKey holds the encrypted info, NOT the plain text.
    if err := db.First(&data, id).Error; err != nil {
        return "", err
    }

    // Check for nil (in case the DB value is NULL)
    if data.ApiKey == nil {
        return "", nil
    }

    // Step 2: Call .String() to decrypt
    // You must pass the context to handle timeouts
    plainText, err := data.ApiKey.String(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt: %w", err)
    }

    return plainText, nil
}
```