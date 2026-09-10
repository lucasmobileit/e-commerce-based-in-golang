// Estado local da aplicação
let allProducts = [];

function getCurrentUserId() {
  const input = document.getElementById("userIdInput");
  return input && input.value.trim() ? input.value.trim() : "guest-default";
}

function getCategoryIcon(category) {
  const map = {
    "Eletrônicos": "💻",
    "Eletrodomésticos": "❄️",
    "Móveis": "🛋️",
    "Sapatos": "👟",
    "Acessórios": "⌚"
  };
  return map[category] || "📦";
}

// 1. Carregar catálogo de produtos (GET /products)
async function fetchProducts() {
  const grid = document.getElementById("productsGrid");
  try {
    const res = await fetch("/products");
    if (!res.ok) throw new Error("Erro na resposta da API");
    allProducts = await res.json();
    renderProducts(allProducts);
  } catch (err) {
    console.error("Falha ao carregar produtos:", err);
    if (grid) grid.innerHTML = "<p style='color:#c00;'>Erro ao conectar com a API ou banco vazio.</p>";
  }
}

// 2. Renderizar cards na tela
function renderProducts(products) {
  const grid = document.getElementById("productsGrid");
  if (!grid) return;
  grid.innerHTML = "";

  if (!products || products.length === 0) {
    grid.innerHTML = "<p>Nenhum produto cadastrado no banco.</p>";
    return;
  }

  products.forEach(p => {
    const card = document.createElement("div");
    card.className = "product-card";
    card.innerHTML = `
      <div class="product-img-box">${getCategoryIcon(p.category)}</div>
      <div class="product-info">
        <span class="product-category">${p.category}</span>
        <h3 class="product-title">${p.name}</h3>
        <span class="product-price">R$ ${Number(p.price).toLocaleString("pt-BR", { minimumFractionDigits: 2 })}</span>
        <span class="product-shipping">⚡ Chegará grátis amanhã</span>
        <button class="btn-add" data-id="${p.id}">Adicionar ao carrinho</button>
      </div>
    `;
    grid.appendChild(card);
  });

  document.querySelectorAll(".btn-add").forEach(btn => {
    btn.addEventListener("click", () => {
      const id = parseInt(btn.getAttribute("data-id"), 10);
      addToCart(id);
    });
  });
}

// 3. Adicionar produto ao carrinho (POST /cart)
async function addToCart(productId) {
  const userId = getCurrentUserId();
  try {
    const res = await fetch("/cart", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-User-ID": userId
      },
      body: JSON.stringify({ product_id: productId })
    });

    if (!res.ok) throw new Error("Não foi possível adicionar o produto");
    await fetchCart();
  } catch (err) {
    alert(err.message);
  }
}

// 4. Buscar e exibir dados do carrinho (GET /cart)
async function fetchCart() {
  const userId = getCurrentUserId();
  try {
    const res = await fetch("/cart", {
      headers: { "X-User-ID": userId }
    });
    if (!res.ok) throw new Error("Erro ao consultar carrinho");
    const data = await res.json();

    const badge = document.getElementById("cartCountBadge");
    if (badge) badge.innerText = data.count || 0;
    renderCartDrawer(data);
  } catch (err) {
    console.error("Falha ao buscar carrinho:", err);
  }
}

function renderCartDrawer(cartData) {
  const list = document.getElementById("cartItemsList");
  const total = document.getElementById("cartTotalText");
  if (!list || !total) return;

  list.innerHTML = "";

  if (!cartData.items || cartData.items.length === 0) {
    list.innerHTML = "<p style='color:#777; text-align:center; margin-top:20px;'>Seu carrinho está vazio.</p>";
    total.innerText = "R$ 0,00";
    return;
  }

  cartData.items.forEach(item => {
    const el = document.createElement("div");
    el.className = "cart-item";
    el.innerHTML = `
      <div>
        <strong>${item.name}</strong><br/>
        <small style="color:#777;">${item.category}</small>
      </div>
      <div>R$ ${Number(item.price).toLocaleString("pt-BR", { minimumFractionDigits: 2 })}</div>
    `;
    list.appendChild(el);
  });

  total.innerText = `R$ ${Number(cartData.total).toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`;
}

// 5. Checkout com pagamento fake e persistência (POST /checkout)
async function checkout() {
  const userId = getCurrentUserId();
  try {
    const res = await fetch("/checkout", {
      method: "POST",
      headers: { "X-User-ID": userId }
    });

    const data = await res.json();

    if (res.status === 201) {
      alert(`✅ SUCESSO! Pedido #${data.order_id} aprovado.\nTotal: R$ ${data.total.toFixed(2)}`);
      await fetchCart();
      document.getElementById("cartModal").classList.add("hidden");
    } else if (res.status === 402) {
      alert(`❌ PAGAMENTO RECUSADO!\nPedido #${data.order_id} registrado como falha.\nOs produtos continuam no seu carrinho.`);
      await fetchCart();
    } else {
      alert("Aviso: " + (data.message || "Erro no processamento"));
    }
  } catch (err) {
    alert("Erro de comunicação com o servidor: " + err.message);
  }
}

// 6. Consultar histórico de pedidos no banco (GET /orders)
async function fetchOrders() {
  const userId = getCurrentUserId();
  const label = document.getElementById("ordersUserLabel");
  if (label) label.innerText = userId;

  try {
    const res = await fetch("/orders", {
      headers: { "X-User-ID": userId }
    });
    const orders = await res.json();
    const list = document.getElementById("ordersList");
    if (!list) return;

    list.innerHTML = "";

    if (!orders || orders.length === 0) {
      list.innerHTML = "<p style='color:#777;'>Nenhum pedido registrado para este usuário.</p>";
    } else {
      orders.forEach(o => {
        const isSuccess = o.status === "completed";
        const badgeColor = isSuccess ? "#00a650" : "#dc3545";
        const badgeText = isSuccess ? "Aprovado" : "Falhou";

        const div = document.createElement("div");
        div.className = "order-entry";
        div.innerHTML = `
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <strong>Pedido #${o.id}</strong>
            <span style="background:${badgeColor}; color:#fff; padding:2px 8px; border-radius:4px; font-size:12px; font-weight:bold;">
              ${badgeText}
            </span>
          </div>
          <small style="color:#777;">${new Date(o.created_at).toLocaleString("pt-BR")}</small><br/>
          <span>Itens: ${o.items ? o.items.length : 0}</span><br/>
          <strong style="color:#333;">Total: R$ ${Number(o.total).toFixed(2)}</strong>
        `;
        list.appendChild(div);
      });
    }

    document.getElementById("ordersModal").classList.remove("hidden");
  } catch (err) {
    alert("Erro ao buscar histórico: " + err.message);
  }
}

// 7. Consultar Métricas do Sistema (GET /metrics)
async function fetchMetrics() {
  try {
    const res = await fetch("/metrics");
    if (!res.ok) throw new Error("Falha ao obter métricas da API");
    const m = await res.json();
    const content = document.getElementById("metricsContent");
    if (!content) return;

    content.innerHTML = `
      <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
        <div style="background: #f8f9fa; padding: 12px; border-radius: 6px; border: 1px solid #e9ecef;">
          <small style="color:#666;">Total de Pedidos</small>
          <h3 style="margin-top:4px; font-size:22px;">${m.total_orders}</h3>
        </div>
        <div style="background: #e8f5e9; padding: 12px; border-radius: 6px; border: 1px solid #c8e6c9;">
          <small style="color:#2e7d32;">Receita Aprovada</small>
          <h3 style="margin-top:4px; font-size:22px; color:#2e7d32;">R$ ${Number(m.revenue).toFixed(2)}</h3>
        </div>
        <div style="background: #ffebee; padding: 12px; border-radius: 6px; border: 1px solid #ffcdd2;">
          <small style="color:#c62828;">Checkouts Falhos</small>
          <h3 style="margin-top:4px; font-size:22px; color:#c62828;">${m.failed_sales}</h3>
        </div>
        <div style="background: #f8f9fa; padding: 12px; border-radius: 6px; border: 1px solid #e9ecef;">
          <small style="color:#666;">Ticket Médio</small>
          <h3 style="margin-top:4px; font-size:22px;">R$ ${Number(m.average_ticket).toFixed(2)}</h3>
        </div>
      </div>
      <div style="margin-top: 14px; background: #e3f2fd; padding: 12px; border-radius: 6px; text-align: center; border: 1px solid #bbdefb;">
        <span style="color:#1565c0;">Taxa de Conversão:</span>
        <strong style="color:#0d47a1; font-size:18px; margin-left: 6px;">${Number(m.conversion_rate).toFixed(1)}%</strong>
      </div>
    `;

    document.getElementById("metricsModal").classList.remove("hidden");
  } catch (err) {
    alert("Erro ao buscar métricas: " + err.message);
  }
}

// 8. Inicialização de eventos
document.addEventListener("DOMContentLoaded", () => {
  fetchProducts();
  fetchCart();

  // Filtro de pesquisa no front
  const searchInput = document.getElementById("searchInput");
  if (searchInput) {
    searchInput.addEventListener("input", (e) => {
      const term = e.target.value.toLowerCase();
      const filtered = allProducts.filter(p =>
        p.name.toLowerCase().includes(term) || p.category.toLowerCase().includes(term)
      );
      renderProducts(filtered);
    });
  }

  // Troca de identificador de usuário
  const userInput = document.getElementById("userIdInput");
  if (userInput) {
    userInput.addEventListener("change", () => {
      fetchCart();
    });
  }

  // Controles do Carrinho
  const btnCart = document.getElementById("btnCart");
  const btnCloseCart = document.getElementById("btnCloseCart");
  const btnCheckout = document.getElementById("btnCheckout");

  if (btnCart) btnCart.onclick = () => document.getElementById("cartModal").classList.remove("hidden");
  if (btnCloseCart) btnCloseCart.onclick = () => document.getElementById("cartModal").classList.add("hidden");
  if (btnCheckout) btnCheckout.onclick = checkout;

  // Controles de Pedidos
  const btnOrders = document.getElementById("btnOrders");
  const btnCloseOrders = document.getElementById("btnCloseOrders");

  if (btnOrders) btnOrders.onclick = fetchOrders;
  if (btnCloseOrders) btnCloseOrders.onclick = () => document.getElementById("ordersModal").classList.add("hidden");

  // Controles do Modal de Métricas
  const btnMetrics = document.getElementById("btnMetrics");
  const btnCloseMetrics = document.getElementById("btnCloseMetrics");

  if (btnMetrics) btnMetrics.onclick = fetchMetrics;
  if (btnCloseMetrics) btnCloseMetrics.onclick = () => document.getElementById("metricsModal").classList.add("hidden");
});
