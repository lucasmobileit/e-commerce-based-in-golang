// Estado local da UI
let allProducts = [];

// Helper para obter o userID atual do campo de texto
function getCurrentUserId() {
  return document.getElementById("userIdInput").value.trim() || "guest-default";
}

// Ícones dinâmicos baseados na categoria
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

// 1. Carregar produtos do backend (GET /products)
async function fetchProducts() {
  try {
    const res = await fetch("/products");
    if (!res.ok) throw new Error("Falha ao buscar produtos");
    allProducts = await res.json();
    renderProducts(allProducts);
  } catch (err) {
    console.error(err);
    document.getElementById("productsGrid").innerHTML = "<p>Erro ao conectar com a API.</p>";
  }
}

// Renderizar cards de produtos
function renderProducts(products) {
  const grid = document.getElementById("productsGrid");
  grid.innerHTML = "";

  if (products.length === 0) {
    grid.innerHTML = "<p>Nenhum produto encontrado.</p>";
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
        <span class="product-price">R$ ${p.price.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}</span>
        <span class="product-shipping">⚡ Chegará grátis amanhã</span>
        <button class="btn-add" onclick="addToCart(${p.id})">Adicionar ao carrinho</button>
      </div>
    `;
    grid.appendChild(card);
  });
}

// 2. Adicionar ao carrinho (POST /cart)
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

    if (!res.ok) throw new Error("Erro ao adicionar produto");
    await fetchCart(); // Atualiza a contagem visual
  } catch (err) {
    alert("Erro ao adicionar produto: " + err.message);
  }
}

// 3. Buscar dados do carrinho (GET /cart)
async function fetchCart() {
  const userId = getCurrentUserId();

  try {
    const res = await fetch("/cart", {
      headers: { "X-User-ID": userId }
    });
    if (!res.ok) throw new Error("Erro ao buscar carrinho");
    const data = await res.json();

    document.getElementById("cartCountBadge").innerText = data.count || 0;
    renderCartDrawer(data);
  } catch (err) {
    console.error(err);
  }
}

// Renderizar gaveta do carrinho
function renderCartDrawer(cartData) {
  const list = document.getElementById("cartItemsList");
  const total = document.getElementById("cartTotalText");
  list.innerHTML = "";

  if (!cartData.items || cartData.items.length === 0) {
    list.innerHTML = "<p style='color:#777;'>Seu carrinho está vazio.</p>";
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
      <div>R$ ${item.price.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}</div>
    `;
    list.appendChild(el);
  });

  total.innerText = `R$ ${cartData.total.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`;
}

// 4. Fechar Compra (POST /checkout)
async function checkout() {
  const userId = getCurrentUserId();

  try {
    const res = await fetch("/checkout", {
      method: "POST",
      headers: { "X-User-ID": userId }
    });

    if (!res.ok) {
      const errText = await res.text();
      throw new Error(errText);
    }

    const order = await res.json();
    alert(`Pedido #${order.id} realizado com sucesso no valor de R$ ${order.total.toFixed(2)}!`);

    await fetchCart();
    document.getElementById("cartModal").classList.add("hidden");
  } catch (err) {
    alert("Falha no checkout: " + err.message);
  }
}

// 5. Histórico de Pedidos (GET /orders)
async function fetchOrders() {
  const userId = getCurrentUserId();
  document.getElementById("ordersUserLabel").innerText = userId;

  try {
    const res = await fetch("/orders", {
      headers: { "X-User-ID": userId }
    });
    const orders = await res.json();
    const list = document.getElementById("ordersList");
    list.innerHTML = "";

    if (orders.length === 0) {
      list.innerHTML = "<p>Nenhum pedido realizado por este usuário.</p>";
    } else {
      orders.forEach(o => {
        const d = new Date(o.created_at);
        const div = document.createElement("div");
        div.className = "order-entry";
        div.innerHTML = `
          <strong>Pedido #${o.id}</strong> - <small>${d.toLocaleString("pt-BR")}</small><br/>
          <span>Itens: ${o.items.length} produto(s)</span><br/>
          <strong style="color:#00a650;">Total: R$ ${o.total.toFixed(2)}</strong>
        `;
        list.appendChild(div);
      });
    }

    document.getElementById("ordersModal").classList.remove("hidden");
  } catch (err) {
    alert("Erro ao buscar pedidos: " + err.message);
  }
}

// Event Listeners e Inicialização
document.addEventListener("DOMContentLoaded", () => {
  fetchProducts();
  fetchCart();

  // Busca instantânea no front
  document.getElementById("searchInput").addEventListener("input", (e) => {
    const term = e.target.value.toLowerCase();
    const filtered = allProducts.filter(p => 
      p.name.toLowerCase().includes(term) || p.category.toLowerCase().includes(term)
    );
    renderProducts(filtered);
  });

  // Troca de Usuário (recarrega estado)
  document.getElementById("userIdInput").addEventListener("change", () => {
    fetchCart();
  });

  // Modais
  document.getElementById("btnCart").onclick = () => document.getElementById("cartModal").classList.remove("hidden");
  document.getElementById("btnCloseCart").onclick = () => document.getElementById("cartModal").classList.add("hidden");
  document.getElementById("btnCheckout").onclick = checkout;

  document.getElementById("btnOrders").onclick = fetchOrders;
  document.getElementById("btnCloseOrders").onclick = () => document.getElementById("ordersModal").classList.add("hidden");
});