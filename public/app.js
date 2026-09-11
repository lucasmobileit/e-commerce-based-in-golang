let allProducts = [];
let appliedCoupon = null;
let currentAuthUser = null;

// 1. Valida a sessão via Cookie no backend
async function checkAuthSession() {
  try {
    const res = await fetch("/auth/me");
    if (!res.ok) {
      window.location.href = "/login.html";
      return false;
    }
    currentAuthUser = await res.json();
    
    // Atualiza o nome exibido no topo da loja
    const label = document.getElementById("userNameLabel");
    if (label && currentAuthUser.name) {
      label.innerText = currentAuthUser.name.split(" ")[0];
    }
    return true;
  } catch (err) {
    window.location.href = "/login.html";
    return false;
  }
}

// 2. Ícones dinâmicos por categoria
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

// 3. Carregar catálogo de produtos (GET /products)
async function fetchProducts() {
  const grid = document.getElementById("productsGrid");
  try {
    const res = await fetch("/products");
    if (!res.ok) throw new Error("Erro na resposta da API");
    allProducts = await res.json();
    renderProducts(allProducts);
  } catch (err) {
    console.error("Falha ao carregar produtos:", err);
    if (grid) grid.innerHTML = "<p style='color:#c00;'>Erro ao carregar produtos.</p>";
  }
}

// 4. Renderizar catálogo na tela com estoque dinâmico
function renderProducts(products) {
  const grid = document.getElementById("productsGrid");
  if (!grid) return;
  grid.innerHTML = "";

  if (!products || products.length === 0) {
    grid.innerHTML = "<p>Nenhum produto cadastrado no banco.</p>";
    return;
  }

  products.forEach(p => {
    const isOutOfStock = p.stock <= 0;
    const stockBadge = isOutOfStock
      ? `<span style="color:#d32f2f; font-weight:bold; font-size:12px;">Esgotado</span>`
      : `<span style="color:#2e7d32; font-weight:bold; font-size:12px;">Disponível: ${p.stock} un.</span>`;

    const buttonHtml = isOutOfStock
      ? `<button class="btn-add" disabled style="background:#ccc; cursor:not-allowed;">Indisponível</button>`
      : `<button class="btn-add" data-id="${p.id}">Adicionar ao carrinho</button>`;

    const card = document.createElement("div");
    card.className = "product-card";
    card.innerHTML = `
      <div class="product-img-box">${getCategoryIcon(p.category)}</div>
      <div class="product-info">
        <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:4px;">
          <span class="product-category">${p.category}</span>
          ${stockBadge}
        </div>
        <h3 class="product-title">${p.name}</h3>
        <span class="product-price">R$ ${Number(p.price).toLocaleString("pt-BR", { minimumFractionDigits: 2 })}</span>
        <span class="product-shipping">⚡ Chegará grátis amanhã</span>
        ${buttonHtml}
      </div>
    `;
    grid.appendChild(card);
  });

  document.querySelectorAll(".btn-add:not([disabled])").forEach(btn => {
    btn.addEventListener("click", () => {
      const id = parseInt(btn.getAttribute("data-id"), 10);
      addToCart(id);
    });
  });
}

// 5. Adicionar ao carrinho (POST /cart)
async function addToCart(productId) {
  try {
    const res = await fetch("/cart", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ product_id: productId })
    });

    if (!res.ok) {
      const errText = await res.text();
      throw new Error(errText);
    }
    await fetchCart();
  } catch (err) {
    alert("Aviso: " + err.message);
  }
}

// 6. Consultar carrinho do usuário logado (GET /cart)
async function fetchCart() {
  try {
    const res = await fetch("/cart");
    if (!res.ok) throw new Error("Erro ao consultar carrinho");
    const data = await res.json();

    const badge = document.getElementById("cartCountBadge");
    if (badge) badge.innerText = data.count || 0;
    renderCartDrawer(data);
  } catch (err) {
    console.error("Falha ao buscar carrinho:", err);
  }
}

// 7. Renderizar drawer do carrinho agrupando quantidades
function renderCartDrawer(cartData) {
  const list = document.getElementById("cartItemsList");
  const subtotalEl = document.getElementById("cartSubtotalText");
  const discountRow = document.getElementById("discountRow");
  const discountEl = document.getElementById("cartDiscountText");
  const discountTag = document.getElementById("discountTag");
  const totalEl = document.getElementById("cartTotalText");

  if (!list || !totalEl) return;
  list.innerHTML = "";

  if (!cartData.items || cartData.items.length === 0) {
    list.innerHTML = "<p style='color:#777; text-align:center; margin-top:20px;'>Seu carrinho está vazio.</p>";
    subtotalEl.innerText = "R$ 0,00";
    totalEl.innerText = "R$ 0,00";
    discountRow.style.display = "none";
    appliedCoupon = null;
    return;
  }

  const grouped = {};
  cartData.items.forEach(item => {
    if (!grouped[item.id]) {
      grouped[item.id] = { ...item, qty: 1 };
    } else {
      grouped[item.id].qty++;
    }
  });

  let subtotal = 0;
  Object.values(grouped).forEach(item => {
    const itemTotal = item.price * item.qty;
    subtotal += itemTotal;

    const el = document.createElement("div");
    el.className = "cart-item";
    el.innerHTML = `
      <div style="flex:1;">
        <strong>${item.name}</strong><br/>
        <span style="display:inline-block; margin-top:2px; background:#e8f0fe; color:#1967d2; padding:1px 6px; border-radius:4px; font-size:12px; font-weight:bold;">
          Qtd: ${item.qty} un.
        </span>
        <small style="color:#777; margin-left: 6px;">(R$ ${item.price.toFixed(2)} un)</small>
      </div>
      <div style="font-weight:bold;">R$ ${itemTotal.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}</div>
    `;
    list.appendChild(el);
  });

  let discount = 0;
  if (appliedCoupon) {
    if (appliedCoupon.discount_type === "percentage") {
      discount = subtotal * (appliedCoupon.discount_value / 100.0);
    } else if (appliedCoupon.discount_type === "fixed") {
      discount = appliedCoupon.discount_value;
    }
    if (discount > subtotal) discount = subtotal;

    discountRow.style.display = "flex";
    discountTag.innerText = appliedCoupon.code;
    discountEl.innerText = `- R$ ${discount.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`;
  } else {
    discountRow.style.display = "none";
  }

  const finalTotal = subtotal - discount;
  subtotalEl.innerText = `R$ ${subtotal.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`;
  totalEl.innerText = `R$ ${finalTotal.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`;
}

// 8. Aplicar cupom promocional (POST /coupons/apply)
async function applyCoupon() {
  const input = document.getElementById("couponInput");
  const status = document.getElementById("couponStatusText");
  const code = input.value.trim().toUpperCase();

  if (!code) {
    status.style.display = "block";
    status.style.color = "#d32f2f";
    status.innerText = "Digite um código de cupom.";
    return;
  }

  try {
    const res = await fetch("/coupons/apply", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code })
    });

    if (!res.ok) {
      appliedCoupon = null;
      status.style.display = "block";
      status.style.color = "#d32f2f";
      status.innerText = "Cupom inválido ou expirado.";
      await fetchCart();
      return;
    }

    appliedCoupon = await res.json();
    status.style.display = "block";
    status.style.color = "#00a650";

    const descText = appliedCoupon.discount_type === "percentage"
      ? `${appliedCoupon.discount_value}% de desconto`
      : `R$ ${appliedCoupon.discount_value.toFixed(2)} de desconto`;

    status.innerText = `✅ Cupom ${appliedCoupon.code} aplicado: ${descText}!`;
    await fetchCart();
  } catch (err) {
    alert("Erro na requisição: " + err.message);
  }
}

// 9. Checkout transacional (POST /checkout)
async function checkout() {
  const couponCode = appliedCoupon ? appliedCoupon.code : "";

  try {
    const res = await fetch("/checkout", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ coupon_code: couponCode })
    });

    const data = await res.json();

    if (res.status === 201) {
      let couponMsg = data.coupon_applied ? `\nCupom: ${data.coupon_applied}` : "";
      alert(`✅ SUCESSO! Pedido #${data.order_id} aprovado.\nTotal: R$ ${data.total.toFixed(2)}${couponMsg}\nEstoque atualizado.`);
      appliedCoupon = null;
      document.getElementById("couponInput").value = "";
      document.getElementById("couponStatusText").style.display = "none";
      await fetchCart();
      await fetchProducts();
      document.getElementById("cartModal").classList.add("hidden");
    } else if (res.status === 402) {
      alert(`❌ PAGAMENTO RECUSADO!\nPedido #${data.order_id} registrado como falha.\nO estoque não foi debitado e seus itens continuam no carrinho.`);
      await fetchCart();
    } else if (res.status === 409) {
      alert(`⚠️ CONFLITO DE ESTOQUE!\n${data.message}`);
      await fetchProducts();
    } else {
      alert("Aviso: " + (data.message || "Erro no processamento"));
    }
  } catch (err) {
    alert("Erro de comunicação: " + err.message);
  }
}

// 10. Histórico de pedidos do usuário autenticado (GET /orders)
async function fetchOrders() {
  const label = document.getElementById("ordersUserLabel");
  if (label && currentAuthUser) {
    label.innerText = currentAuthUser.name || currentAuthUser.id;
  }

  try {
    const res = await fetch("/orders");
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
        const couponInfo = o.coupon_applied ? `<span style="color:#00a650; font-weight:bold;">[Cupom: ${o.coupon_applied}]</span>` : "";

        const div = document.createElement("div");
        div.className = "order-entry";
        div.innerHTML = `
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <strong>Pedido #${o.id} ${couponInfo}</strong>
            <span style="background:${badgeColor}; color:#fff; padding:2px 8px; border-radius:4px; font-size:12px; font-weight:bold;">
              ${badgeText}
            </span>
          </div>
          <small style="color:#777;">${new Date(o.created_at).toLocaleString("pt-BR")}</small><br/>
          <span>Subtotal: R$ ${Number(o.subtotal || o.total).toFixed(2)} | Desconto: R$ ${Number(o.discount || 0).toFixed(2)}</span><br/>
          <strong style="color:#333;">Total Final: R$ ${Number(o.total).toFixed(2)}</strong>
        `;
        list.appendChild(div);
      });
    }

    document.getElementById("ordersModal").classList.remove("hidden");
  } catch (err) {
    alert("Erro ao buscar histórico: " + err.message);
  }
}

// 11. Inicialização de Eventos
document.addEventListener("DOMContentLoaded", async () => {
  const authenticated = await checkAuthSession();
  if (!authenticated) return;

  fetchProducts();
  fetchCart();

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

  document.getElementById("btnCart").onclick = () => document.getElementById("cartModal").classList.remove("hidden");
  document.getElementById("btnCloseCart").onclick = () => document.getElementById("cartModal").classList.add("hidden");
  document.getElementById("btnCheckout").onclick = checkout;
  document.getElementById("btnApplyCoupon").onclick = applyCoupon;

  document.getElementById("btnOrders").onclick = fetchOrders;
  document.getElementById("btnCloseOrders").onclick = () => document.getElementById("ordersModal").classList.add("hidden");
});

const btnLogout = document.getElementById("btnLogout");
if (btnLogout) {
  btnLogout.onclick = async () => {
    await fetch("/logout", { method: "POST" });
    window.location.href = "/login.html";
  };
}
