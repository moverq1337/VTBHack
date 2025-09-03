package utils

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

func UploadToYandexDisk(filePath, fileName string) (string, error) {
	oauthToken := os.Getenv("YANDEX_DISK_TOKEN") // token
	client := &http.Client{}
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("PUT", "https://cloud-api.yandex.net/v1/disk/resources/upload?path=/hr-ai/"+fileName, bytes.NewReader(fileData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "OAuth "+oauthToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("ошибка загрузки: %v", resp.Status)
	}

	return "https://disk.yandex.ru/d/" + fileName, nil
}
