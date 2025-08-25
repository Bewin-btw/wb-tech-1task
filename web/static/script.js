function getOrder() {
    const orderUid = document.getElementById('orderUid').value.trim();
    if (!orderUid) {
        showError('Please enter Order UID');
        return;
    }

    showLoading();

    fetch(`http://localhost:8080/order?uid=${encodeURIComponent(orderUid)}`)
        .then(response => {
            if (!response.ok) {
                if (response.status === 404) {
                    throw new Error('Order not found');
                } else if (response.status === 400) {
                    throw new Error('Invalid Order UID');
                } else {
                    throw new Error('Server error: ' + response.status);
                }
            }
            return response.json();
        })
        .then(data => {
            displayOrder(data);
        })
        .catch(error => {
            showError('Error: ' + error.message);
        });
}

function showLoading() {
    document.getElementById('orderResult').innerHTML = `
        <div class="loading">Loading...</div>
    `;
}

function showError(message) {
    document.getElementById('orderResult').innerHTML = `
        <div class="error">${message}</div>
    `;
}

function displayOrder(order) {
    const orderHTML = `
        <div class="order-info">
            <h2>Order Information</h2>
            <div class="order-details">
                <p><strong>Order UID:</strong> ${order.order_uid}</p>
                <p><strong>Track Number:</strong> ${order.track_number}</p>
                <p><strong>Entry:</strong> ${order.entry}</p>
                <p><strong>Customer ID:</strong> ${order.customer_id}</p>
                <p><strong>Delivery Service:</strong> ${order.delivery_service}</p>
                <p><strong>Date Created:</strong> ${order.date_created}</p>
            </div>
            
            <div class="section">
                <div class="section-title">Delivery Information</div>
                <p><strong>Name:</strong> ${order.delivery.name}</p>
                <p><strong>Phone:</strong> ${order.delivery.phone}</p>
                <p><strong>Zip:</strong> ${order.delivery.zip}</p>
                <p><strong>City:</strong> ${order.delivery.city}</p>
                <p><strong>Address:</strong> ${order.delivery.address}</p>
                <p><strong>Region:</strong> ${order.delivery.region}</p>
                <p><strong>Email:</strong> ${order.delivery.email}</p>
            </div>
            
            <div class="section">
                <div class="section-title">Payment Information</div>
                <p><strong>Transaction:</strong> ${order.payment.transaction}</p>
                <p><strong>Currency:</strong> ${order.payment.currency}</p>
                <p><strong>Provider:</strong> ${order.payment.provider}</p>
                <p><strong>Amount:</strong> ${order.payment.amount}</p>
                <p><strong>Payment Date:</strong> ${new Date(order.payment.payment_dt * 1000).toLocaleString()}</p>
                <p><strong>Bank:</strong> ${order.payment.bank}</p>
                <p><strong>Delivery Cost:</strong> ${order.payment.delivery_cost}</p>
                <p><strong>Goods Total:</strong> ${order.payment.goods_total}</p>
            </div>
            
            <div class="section">
                <div class="section-title">Items (${order.items.length})</div>
                ${order.items.map(item => `
                    <div class="item">
                        <p><strong>Name:</strong> ${item.name}</p>
                        <p><strong>Brand:</strong> ${item.brand}</p>
                        <p><strong>Price:</strong> ${item.price}</p>
                        <p><strong>Total Price:</strong> ${item.total_price}</p>
                        <p><strong>Sale:</strong> ${item.sale}%</p>
                        <p><strong>Size:</strong> ${item.size}</p>
                        <p><strong>Status:</strong> ${item.status}</p>
                    </div>
                `).join('')}
            </div>
        </div>
    `;

    document.getElementById('orderResult').innerHTML = orderHTML;
}

// Добавляем обработчик нажатия Enter в поле ввода
document.getElementById('orderUid').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        getOrder();
    }
});