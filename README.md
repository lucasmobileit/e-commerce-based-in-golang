# e-commerce-based-in-golang
Simulação de um e-commerce inspirado no Mercado Livre

# Arquitetura

               Mercado Go
                   │
            HTML / CSS / JS
                   │
             HTTP REST
                   ▼
              Backend Go
             ┌─────┴─────┐
             │           │
          Carrinho    Checkout
                         │
                  Fake Payment
                         │
                         ▼
                      SQLite
                  products
                  orders
                  order_items
                         │
                         ▼
                   GET /metrics
