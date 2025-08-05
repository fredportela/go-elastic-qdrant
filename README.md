
# 🔁 Elasticsearch → Qdrant Exporter

Este projeto em Go realiza a exportação de documentos do Elasticsearch para o Qdrant, convertendo os textos em vetores (sparse ou dense embeddings) e armazenando-os como pontos vetoriais em uma coleção do Qdrant.

---

## 🧩 Funcionalidades

- Conecta-se a um cluster Elasticsearch com autenticação básica
- Realiza consultas paginadas com `match_all`
- Extrai os campos `id` e `texto` dos documentos
- Gera embeddings (simulados no exemplo)
- Cria uma coleção no Qdrant (se não existir)
- Insere os documentos como pontos vetoriais na coleção
- Controla e exibe logs de progresso e erros

---

## 📦 Requisitos

- Go 1.22+ instalado
- Elasticsearch com índice e documentos acessíveis
- Qdrant rodando localmente (porta padrão: `6334`) ou em outro host configurado
- Permissões adequadas para leitura no Elasticsearch e escrita no Qdrant

---

## 🛠️ Instalação

Clone este repositório:

```bash
git clone https://github.com/fredportela/go-elastic-qdrant
cd go-elastic-qdrant
```

Instale as dependências:

```bash
go mod tidy
```

---

## ⚙️ Configuração

Edite as constantes no topo do arquivo `main.go` conforme o seu ambiente:

```go
const (
    esURL          = "https://elastic:9200/index/_search"  // URL do Elasticsearch
    username       = "usuario_elastic"                     // Usuário ES
    password       = "senha_elastic"                       // Senha ES
    pageSize       = 1000                                  // Tamanho dos lotes de busca
    collectionName = "nome_collection_qdrant"              // Nome da coleção Qdrant
    vectorSize     = 1536                                  // Tamanho dos embeddings
    qdrantHost     = "localhost"                           // Host Qdrant
    qdrantPort     = 6334                                  // Porta Qdrant
)
```

---

## ▶️ Execução

Execute o programa com:

```bash
go run main.go
```

Durante a execução, o programa irá:

- Criar a coleção no Qdrant (se necessário)
- Ler documentos do Elasticsearch
- Inserir no Qdrant como pontos vetoriais
- Exibir logs com sucesso ou falha de inserção

---

## 🧠 Embedding

Neste exemplo, a função `generateEmbedding()` retorna um vetor zerado ou esparso simulado. Para uso real com modelos como OpenAI, HuggingFace, Cohere, etc., substitua esta função:

```go
func generateEmbedding(texto string) []float32 {
    // Faça chamada real à API de embeddings aqui
    return embedding
}
```

Ou integre um modelo local como o [Instructor](https://github.com/jina-ai/instructor) ou [BGE](https://huggingface.co/BAAI/bge-small-en).

---

## 💡 Exemplo de Documento Esperado

Seu índice do Elasticsearch deve conter documentos com pelo menos os campos:

```json
{
  "id": 123456,
  "texto": "Este é o conteúdo do documento legal ou normativo."
}
```

---

## 🧪 Testando com Elasticsearch Local

Execute o Elasticsearch local com Docker:

```bash
docker run -d --name elastic -p 9200:9200 -e "discovery.type=single-node" -e "xpack.security.enabled=false" elasticsearch:8.13.4
```

Depois, insira alguns documentos de teste com `curl` ou Postman.

---

## 📦 Dependências

- [qdrant/go-client](https://github.com/qdrant/go-client) – cliente oficial Go para Qdrant
- `net/http`, `encoding/json`, `crypto/tls` – bibliotecas padrão Go

---

## 📈 Logs

A saída do programa fornece informações detalhadas:

- Conexão e criação de coleção
- Quantidade de documentos processados por lote
- Erros de conexão, leitura ou inserção

---

## 🧹 Limpeza (opcional)

Para apagar a coleção do Qdrant após testes:

```bash
curl -X DELETE "http://localhost:6334/collections/nome_collection_qdrant"
```

---

## 📄 Licença

Este projeto é fornecido sob a licença MIT.

---

## 🤝 Contribuindo

Pull requests são bem-vindos. Para grandes alterações, abra uma *issue* antes para discutir o que você deseja modificar.
