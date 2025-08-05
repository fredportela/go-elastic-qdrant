package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"github.com/qdrant/go-client/qdrant"
)

const (
	esURL          = "https://elastic:9200/index/_search"
	username       = "usuario_elastic"
	password       = "senha_elastic"
	pageSize       = 1000
	// Configurações Qdrant
	collectionName = "nome_collection_qdrant"
	vectorSize     = 1536
	qdrantHost = "localhost"
	qdrantPort = 6334
)

// Estruturas para resposta do Elasticsearch
type Hit struct {
	Source map[string]interface{} `json:"_source"`
}

type HitsContainer struct {
	Total struct {
		Value int `json:"value"`
	} `json:"total"`
	Hits []Hit `json:"hits"`
}

type SearchResponse struct {
	Hits HitsContainer `json:"hits"`
}

// Estrutura para dados do documento
type DocumentData struct {
	ID     uint64
	Texto  string
}

// Cliente personalizado para Elasticsearch
type ElasticsearchClient struct {
	httpClient *http.Client
}

// Cliente personalizado para Qdrant
type QdrantClient struct {
	client *qdrant.Client
}

func NewElasticsearchClient() *ElasticsearchClient {
	return &ElasticsearchClient{
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 10 * time.Second,
		},
	}
}

func NewQdrantClient() (*QdrantClient, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: qdrantHost,
		Port: qdrantPort,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar com Qdrant: %v", err)
	}

	return &QdrantClient{
		client: client,
	}, nil
}

func (qc *QdrantClient) Close() error {
	return qc.client.Close()
}

func (ec *ElasticsearchClient) searchDocuments(from int) (*SearchResponse, error) {
	query := fmt.Sprintf(`{
		"size": %d,
		"from": %d,
		"track_total_hits": true,
		"_source": [
			"id",
			"texto"
		],
		"query": {
			"match_all": {}
		}
	}`, pageSize, from)

	req, err := http.NewRequest("POST", esURL, strings.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ec.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return &result, nil
}

func extractDocumentData(hit Hit) DocumentData {
	data := DocumentData{}

	// Extrair ID
	if v, ok := hit.Source["id"].(float64); ok {
		data.ID = uint64(v)
	}

	// Extrair campos de texto
	if v, ok := hit.Source["texto"].(string); ok {
		data.Texto = v
	}

	return data
}

func generateEmbedding(texto string) []float32 {
	embedding := make([]float32, vectorSize)
	return embedding
}

func (qc *QdrantClient) createCollection() error {
	exists, err := qc.client.CollectionExists(context.Background(), collectionName)
	if err != nil {
		return fmt.Errorf("erro ao verificar se coleção existe: %v", err)
	}

	if exists {
		log.Printf("Coleção '%s' já existe", collectionName)
		return nil
	}

	err = qc.client.CreateCollection(context.Background(), &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil {
		return fmt.Errorf("erro ao criar coleção: %v", err)
	}

	log.Printf("Coleção '%s' criada com sucesso", collectionName)
	return nil
}

func (qc *QdrantClient) upsertDocument(doc DocumentData) error {
	// Gerar embedding
	embedding := generateEmbedding(doc.Texto)
	
	// Criar payload
	payload := map[string]interface{}{
		"texto":  doc.Texto,
	}

	// Criar ponto
	point := &qdrant.PointStruct{
		Id: qdrant.NewIDNum(doc.ID),
		Vectors: qdrant.NewVectors(embedding...),
		Payload: qdrant.NewValueMap(payload),
	}

	// Upsert no Qdrant
	_, err := qc.client.Upsert(context.Background(), &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         []*qdrant.PointStruct{point},
	})

	return err
}

func main() {
	log.Println("Iniciando exportação Elasticsearch → Qdrant")

	// Inicializar clientes
	esClient := NewElasticsearchClient()

	qdrantClient, err := NewQdrantClient()
	if err != nil {
		log.Fatalf("Erro ao conectar com Qdrant: %v", err)
	}
	defer qdrantClient.Close()

	// Criar coleção no Qdrant
	log.Println("Criando coleção no Qdrant...")
	if err := qdrantClient.createCollection(); err != nil {
		log.Fatalf("Erro ao criar coleção: %v", err)
	}

	// Processar documentos em lotes
	from := 0
	totalProcessados := 0
	erros := 0

	for {
		log.Printf("Buscando documentos de %d a %d...", from, from+pageSize)

		// Buscar documentos no Elasticsearch
		result, err := esClient.searchDocuments(from)
		if err != nil {
			log.Printf("Erro ao buscar documentos: %v", err)
			erros++
			if erros >= 5 {
				log.Fatal("Muitos erros consecutivos, encerrando")
			}
			continue
		}

		// Se não há mais documentos, encerrar
		if len(result.Hits.Hits) == 0 {
			log.Println("Não há mais documentos para processar")
			break
		}

		log.Printf("Total de documentos encontrados: %d", result.Hits.Total.Value)

		// Processar cada documento
		sucessos := 0
		for i, hit := range result.Hits.Hits {
			doc := extractDocumentData(hit)

			if err := qdrantClient.upsertDocument(doc); err != nil {
				log.Printf("Erro ao inserir documento %d: %v", doc.ID, err)
				erros++
			} else {
				sucessos++
			}

			// Log de progresso a cada 100 documentos
			if (i+1)%100 == 0 {
				log.Printf("Processados %d/%d documentos do lote atual", i+1, len(result.Hits.Hits))
			}
		}

		totalProcessados += sucessos
		from += pageSize

		log.Printf("Lote concluído: %d sucessos, %d erros. Total processado: %d",
			sucessos, erros, totalProcessados)

		// Pequena pausa entre lotes para não sobrecarregar
		time.Sleep(10 * time.Millisecond)
	}

	log.Printf("Exportação finalizada!")
	log.Printf("Total de documentos processados: %d", totalProcessados)
	log.Printf("Total de erros: %d", erros)
}

