function observer() {
	const params = new URLSearchParams(window.location.search);
	const apiBase = (params.get('api') || '').replace(/\/$/, '');
	const logGroup = '/ecs/demo-order-api';

	const TOOL_LABELS = {
		describe_alarm: 'Inspecting alarm',
		query_logs: 'Searching logs',
		get_metric_data: 'Fetching metrics',
		get_xray_traces: 'Analysing traces',
		get_source_file: 'Reading source code',
		create_issue: 'Creating issue',
	};

	return {
		region: params.get('region') || 'eu-west-2',
		repo: params.get('repo') || '',
		view: 'home',
		incidentId: null,
		incidents: [],
		steps: [],
		lastSeq: 0,
		polling: null,
		discoveryPolling: null,
		incidentPolling: null,
		completed: false,
		startTime: null,
		elapsed: '0s',
		elapsedTimer: null,
		fullText: '',
		issueUrl: null,
		watching: true,
		liveIncidentId: null,

		get statusClass() {
			if (this.completed) return 'status-complete';
			if (this.incidentId) return 'status-active';
			return 'status-waiting';
		},

		get statusText() {
			if (this.completed) return 'Complete';
			if (this.incidentId) return 'Investigating';
			return 'Waiting';
		},

		get toolCount() {
			return this.steps.filter(s => s.event_type === 'tool_start').length;
		},

		get toolCountLabel() {
			const total = this.toolCount;
			const done = this.steps.filter(s => s.event_type === 'tool_start' && s.completed).length;
			if (this.completed) return `${total} tools used`;
			return `${done}/${total}`;
		},

		// Tool steps in reverse order (newest first).
		get toolSteps() {
			return [...this.steps.filter(s => s.event_type === 'tool_start')].reverse();
		},

		get renderedMarkdown() {
			if (!this.fullText) return '<p class="text-muted">Waiting for agent output&hellip;</p>';
			try {
				return marked.parse(this.fullText);
			} catch {
				return `<pre>${this.fullText}</pre>`;
			}
		},

		async init() {
			// When served from the observer subdomain (no query params), fetch config.
			if (!params.has('api') && !params.has('region')) {
				try {
					const r = await fetch('/config');
					if (r.ok) {
						const cfg = await r.json();
						if (cfg.region) this.region = cfg.region;
						if (cfg.repo) this.repo = cfg.repo;
					}
				} catch { /* use defaults */ }
			}
			this.fetchIncidents();
			this.incidentPolling = setInterval(() => this.fetchIncidents(), 5000);
			this.startDiscovery();
		},

		// Fetch the incident list for the dropdown.
		async fetchIncidents() {
			try {
				const r = await fetch(`${apiBase}/api/agent-events/incidents`);
				if (!r.ok) return;
				this.incidents = await r.json();
			} catch { /* ignore */ }
		},

		startDiscovery() {
			this.watching = true;
			this.fetchLatest();
			this.discoveryPolling = setInterval(() => this.fetchLatest(), 2000);
		},

		async fetchLatest() {
			try {
				const r = await fetch(`${apiBase}/api/agent-events/latest`);
				if (!r.ok) return;
				const data = await r.json();
				if (data.incident_id && data.incident_id !== this.incidentId) {
					if (this.view === 'home') {
						// Mark this as a live incident so the home view shows it
						this.liveIncidentId = data.incident_id;
						this.fetchIncidents();
					} else {
						this.loadIncident(data.incident_id, true);
						this.fetchIncidents();
					}
				}
			} catch { /* ignore */ }
		},

		// Load a specific incident (either from discovery or dropdown selection).
		loadIncident(id, isLive) {
			clearInterval(this.polling);
			this.polling = null;

			this.incidentId = id;
			this.steps = [];
			this.lastSeq = 0;
			this.completed = false;
			this.fullText = '';
			this.issueUrl = null;

			if (isLive) {
				this.startTime = Date.now();
				this.startElapsedTimer();
			} else {
				clearInterval(this.elapsedTimer);
				this.elapsed = '';
			}

			this.startEventPolling();
		},

		// Open a run from the home view.
		openRun(id) {
			const isLive = id === this.liveIncidentId;
			this.view = 'detail';
			this.loadIncident(id, isLive);
		},

		// Return to the home view.
		goHome() {
			clearInterval(this.polling);
			clearInterval(this.elapsedTimer);
			this.polling = null;
			this.view = 'home';
			this.incidentId = null;
			this.steps = [];
			this.lastSeq = 0;
			this.completed = false;
			this.startTime = null;
			this.elapsed = '';
			this.fullText = '';
			this.issueUrl = null;
			this.fetchIncidents();
			if (!this.discoveryPolling) {
				this.startDiscovery();
			}
		},

		// Select an incident from the dropdown.
		selectIncident(id) {
			if (id === this.incidentId) return;
			// Stop watching for new incidents when manually selecting.
			clearInterval(this.discoveryPolling);
			this.discoveryPolling = null;
			this.watching = false;
			this.loadIncident(id, false);
		},

		// Resume watching for the latest incident.
		watchLatest() {
			if (this.liveIncidentId) {
				this.openRun(this.liveIncidentId);
				return;
			}
			this.goHome();
		},

		relativeTime(iso) {
			if (!iso) return '';
			const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
			if (diff < 5) return 'just now';
			if (diff < 60) return `${diff}s ago`;
			if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
			if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
			return `${Math.floor(diff / 86400)}d ago`;
		},

		cleanAlarmName(name) {
			return name.replace('demo-incident-response-', '');
		},

		startElapsedTimer() {
			clearInterval(this.elapsedTimer);
			this.elapsedTimer = setInterval(() => {
				if (!this.startTime) return;
				const secs = Math.floor((Date.now() - this.startTime) / 1000);
				if (secs < 60) {
					this.elapsed = `${secs}s`;
				} else {
					const m = Math.floor(secs / 60);
					const s = secs % 60;
					this.elapsed = `${m}m ${s}s`;
				}
			}, 1000);
		},

		startEventPolling() {
			this.fetchEvents();
			this.polling = setInterval(() => this.fetchEvents(), 1500);
		},

		async fetchEvents() {
			if (!this.incidentId) return;
			try {
				const url = `${apiBase}/api/agent-events?incident_id=${encodeURIComponent(this.incidentId)}&after=${this.lastSeq}`;
				const r = await fetch(url);
				if (!r.ok) return;
				const events = await r.json();
				if (!events.length) return;

				for (const ev of events) {
					this.processEvent(ev);
					this.lastSeq = Math.max(this.lastSeq, ev.seq);
				}
			} catch { /* ignore */ }
		},

		processEvent(ev) {
			// For tool_end, update the matching tool_start with duration and input.
			if (ev.event_type === 'tool_end') {
				const match = [...this.steps].reverse().find(
					s => s.event_type === 'tool_start' && s.detail.tool === ev.detail.tool && !s.completed
				);
				if (match) {
					match.completed = true;
					match.duration_s = ev.detail.duration_s;
					if (ev.detail.input) {
						match.input = ev.detail.input;
					}
				}
				return;
			}

			// For tool_result, merge result data into the matching tool_start.
			if (ev.event_type === 'tool_result') {
				const match = [...this.steps].reverse().find(
					s => s.event_type === 'tool_start' && s.detail.tool === ev.detail.tool
				);
				if (match && ev.detail.result) {
					match.result = ev.detail.result;
					// Also set issueUrl directly from create_issue result.
					if (ev.detail.tool === 'create_issue' && ev.detail.result.issue_url) {
						this.issueUrl = ev.detail.result.issue_url;
					}
				}
				return;
			}

			// Accumulate all text into a single markdown string.
			if (ev.event_type === 'text') {
				this.fullText += ev.detail.text;
				// Extract issue URL — prefer result data from create_issue tool, fall back to regex.
				if (!this.issueUrl) {
					const createStep = this.steps.find(s => s.detail.tool === 'create_issue' && s.result && s.result.issue_url);
					if (createStep) {
						this.issueUrl = createStep.result.issue_url;
					} else {
						const match = this.fullText.match(/https:\/\/(?:github\.com|gitlab\.com)[^\s)>\]]+\/issues\/\d+/);
						if (match) this.issueUrl = match[0];
					}
				}
				// Auto-scroll output pane.
				this.$nextTick(() => {
					const body = document.querySelector('.pane-output .pane-body');
					if (body) body.scrollTop = body.scrollHeight;
				});
				return;
			}

			if (ev.event_type === 'complete') {
				this.completed = true;
				clearInterval(this.polling);
				clearInterval(this.elapsedTimer);
				this.polling = null;
				if (this.incidentId === this.liveIncidentId) {
					this.liveIncidentId = null;
				}
			}

			// Only push tool_start and complete to steps.
			if (ev.event_type === 'tool_start' || ev.event_type === 'complete') {
				this.steps.push(ev);
			}
		},

		incidentLabel(inc) {
			const d = new Date(inc.started_at);
			const date = d.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
			const time = d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
			const name = inc.alarm_name.replace('demo-incident-response-', '');
			return `${name} — ${date} ${time}`;
		},

		toolLabel(name) {
			return TOOL_LABELS[name] || name;
		},

		formatParam(val) {
			if (typeof val === 'object') return JSON.stringify(val, null, 2);
			return String(val);
		},

		isLongParam(val) {
			return this.formatParam(val).length > 120;
		},

		truncateParam(val) {
			const s = this.formatParam(val);
			return s.slice(0, 120) + '…';
		},

		toolLink(step) {
			const tool = step.detail.tool;
			const input = step.input || {};
			const result = step.result || {};
			const base = `https://${this.region}.console.aws.amazon.com`;

			if (tool === 'describe_alarm') {
				return `${base}/cloudwatch/home?region=${this.region}#alarmsV2:alarm/demo-incident-response-error-rate`;
			}
			if (tool === 'query_logs' && input.query) {
				const q = encodeURIComponent(input.query);
				const lg = encodeURIComponent(logGroup);
				return `${base}/cloudwatch/home?region=${this.region}#logsV2:logs-insights$3FqueryDetail$3D~(source~(~'${lg})~editorString~'${q})`;
			}
			if (tool === 'get_xray_traces') {
				// Link to specific trace if we have trace IDs from the result.
				if (result.trace_ids && result.trace_ids.length > 0) {
					const traceId = result.trace_ids[0];
					return `${base}/xray/home?region=${this.region}#/traces/${traceId}`;
				}
				return `${base}/xray/home?region=${this.region}#/traces`;
			}
			if (tool === 'get_metric_data') {
				return `${base}/cloudwatch/home?region=${this.region}#metricsV2`;
			}
			if (tool === 'get_source_file' && input.file_path && this.repo) {
				return `https://github.com/${this.repo}/blob/main/${input.file_path}`;
			}
			if (tool === 'create_issue' && result.issue_url) {
				return result.issue_url;
			}
			return null;
		},

		queryLink(query) {
			if (!query) return null;
			const base = `https://${this.region}.console.aws.amazon.com`;
			const q = encodeURIComponent(query);
			const lg = encodeURIComponent(logGroup);
			return `${base}/cloudwatch/home?region=${this.region}#logsV2:logs-insights$3FqueryDetail$3D~(source~(~'${lg})~editorString~'${q})`;
		},

		async copyToClipboard(text, event) {
			try {
				await navigator.clipboard.writeText(text);
				const btn = event.currentTarget;
				const orig = btn.textContent;
				btn.textContent = 'Copied!';
				setTimeout(() => btn.textContent = orig, 1500);
			} catch { /* ignore */ }
		},

		formatTime(ts) {
			if (!ts) return '';
			const d = new Date(ts);
			return d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
		},
	};
}
