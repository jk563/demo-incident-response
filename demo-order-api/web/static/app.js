function app() {
	return {
		tab: 'create',
		health: 'loading',

		// Create order
		items: [{ name: '', quantity: 1, unit_price: 0 }],
		discountCode: '',
		creating: false,
		createMsg: '',
		createOk: false,

		// List orders
		orders: [],
		statusFilter: '',
		listMsg: '',

		// Lookup / detail
		lookupId: '',
		lookingUp: false,
		lookupMsg: '',
		detail: null,
		detailMsg: '',
		detailOk: false,
		refunding: false,

		async init() {
			this.checkHealth();
			setInterval(() => this.checkHealth(), 30000);
		},

		async checkHealth() {
			try {
				const r = await fetch('/health');
				this.health = r.ok ? 'ok' : 'err';
			} catch {
				this.health = 'err';
			}
		},

		async createOrder() {
			this.createMsg = '';
			const validItems = this.items.filter(i => i.name.trim() !== '');
			if (validItems.length === 0) {
				this.createMsg = 'Add at least one item with a name.';
				this.createOk = false;
				return;
			}

			this.creating = true;
			try {
				const body = { items: validItems };
				if (this.discountCode) {
					body.discount_code = this.discountCode;
				}

				const r = await fetch('/orders', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(body),
				});

				if (!r.ok) {
					const err = await r.json().catch(() => ({}));
					throw new Error(err.error || `Server returned ${r.status}`);
				}

				const order = await r.json();
				this.createMsg = `Order created: ${order.id}`;
				this.createOk = true;
				this.items = [{ name: '', quantity: 1, unit_price: 0 }];
				this.discountCode = '';
			} catch (e) {
				this.createMsg = e.message;
				this.createOk = false;
			} finally {
				this.creating = false;
			}
		},

		async fetchOrders() {
			this.listMsg = '';
			try {
				const url = this.statusFilter ? `/orders?status=${this.statusFilter}` : '/orders';
				const r = await fetch(url);
				if (!r.ok) throw new Error(`Server returned ${r.status}`);
				this.orders = await r.json();
			} catch (e) {
				this.listMsg = e.message;
			}
		},

		async viewOrder(id) {
			this.tab = 'lookup';
			this.lookupId = id;
			await this.lookupOrder();
		},

		async lookupOrder() {
			if (!this.lookupId.trim()) return;
			this.lookupMsg = '';
			this.detail = null;
			this.detailMsg = '';
			this.lookingUp = true;
			try {
				const r = await fetch(`/orders/${this.lookupId.trim()}`);
				if (r.status === 404) throw new Error('Order not found.');
				if (!r.ok) throw new Error(`Server returned ${r.status}`);
				this.detail = await r.json();
			} catch (e) {
				this.lookupMsg = e.message;
			} finally {
				this.lookingUp = false;
			}
		},

		async refundOrder(id) {
			this.detailMsg = '';
			this.refunding = true;
			try {
				const r = await fetch(`/orders/${id}/refund`, { method: 'POST' });
				if (r.status === 409) throw new Error('Order has already been refunded.');
				if (!r.ok) {
					const err = await r.json().catch(() => ({}));
					throw new Error(err.error || `Server returned ${r.status}`);
				}
				this.detail = await r.json();
				this.detailMsg = 'Order refunded successfully.';
				this.detailOk = true;
			} catch (e) {
				this.detailMsg = e.message;
				this.detailOk = false;
			} finally {
				this.refunding = false;
			}
		},
	};
}
