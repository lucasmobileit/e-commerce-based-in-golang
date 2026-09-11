Aqui está um `README.md` direto ao ponto, técnico e estruturado para quem for clonar e rodar o projeto localmente ou via container:

```markdown
# MercadoGO — E-Commerce Platform

Plataforma de e-commerce inspirada no modelo do Mercado Livre, desenvolvida com backend em **Go (Golang)** e interface web estática (HTML/CSS/JavaScript). Projeto integrador com foco em arquitetura de microsserviços/módulos e futura instrumentação de métricas (KPIs e KRIs).

---

## 🛠️ Tecnologias Utilizadas

- **Linguagem:** Go (Golang 1.22+)
- **Frontend:** HTML5, CSS3, Vanilla JavaScript
- **Rotas e API:** Endpoints REST em Go
- **Arquivos Estáticos:** Servidos diretamente via Go (`/public`)

---

## 📁 Estrutura do Projeto

```text
├── main.go               # Ponto de entrada do servidor HTTP / rotas
├── go.mod                # Gerenciamento de módulos e dependências
├── go.sum                # Checksum das dependências Go
└── public/               # Frontend estático
    ├── index.html        # Vitrine / catálogo de produtos
    ├── login.html        # Autenticação de usuários
    ├── admin.html        # Painel administrativo
    ├── app.js            # Lógica de consumo da API e renderização
    └── style.css         # Estilização da interface

```

---

## 🚀 Como Executar

### Pré-requisitos

* [Go](https://go.dev/dl/) versão **1.22** ou superior instalada.
* [Git](https://git-scm.com/) instalado.

### 1. Clonar o repositório

```bash
git clone [https://github.com/lucasmobileit/e-commerce-based-in-golang.git](https://github.com/lucasmobileit/e-commerce-based-in-golang.git)
cd e-commerce-based-in-golang

```

### 2. Baixar dependências

```bash
go mod tidy
go mod download

```

### 3. Rodar a aplicação

Execute diretamente via terminal:

```bash
go run main.go

```

Ou compile o binário e execute:

```bash
# Compilar
go build -o mercadogo main.go

# Executar (Linux/macOS)
./mercadogo

# Executar (Windows)
.\mercadogo.exe

```

---

## 🌐 Acesso à Aplicação

Por padrão, o servidor subirá na porta local definida no `main.go` (geralmente `:8080` ou `:3000`):

* **Vitrine (Loja):** [http://localhost:8080](http://localhost:8080)
* **Login:** [http://localhost:8080/login.html](http://localhost:8080/login.html)
* **Painel Administrativo:** [http://localhost:8080/admin.html](http://localhost:8080/admin.html)

---

## ⚙️ Variáveis de Ambiente (Opcional)

Caso configure suporte a variáveis de ambiente (`.env`), defina os seguintes parâmetros antes de iniciar:

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `PORT` | `8080` | Porta em que o servidor HTTP irá escutar |
| `ENV` | `development` | Ambiente de execução (`development` / `production`) |

Exemplo de execução alterando a porta:

```bash
PORT=8081 go run main.go

```


```

---

*Nota: Se a porta padrão configurada no seu `main.go` for diferente de `8080` (ex: `3000` ou `8000`), basta trocar o número na seção "Acesso à Aplicação".*

```
