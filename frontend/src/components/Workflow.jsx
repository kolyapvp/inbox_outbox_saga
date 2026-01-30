import React, { useEffect, useMemo, useState } from 'react';

const fmtTime = (iso) => {
  if (!iso) return '';
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return '';
  }
};

const ruService = (s) => {
  const map = {
    'order-service': 'Сервис заказов',
    'payment-service': 'Сервис оплаты',
    'ticket-service': 'Сервис билетов',
    worker: 'Воркер',
    api: 'API',
  };
  return map[s] || s || '—';
};

const ruEventType = (s) => {
  const map = {
    OrderCreated: 'Заказ создан',
    PaymentAuthorized: 'Оплата подтверждена',
    TicketIssued: 'Билет выпущен',
    PaymentFailed: 'Ошибка оплаты',
    RefundInitiated: 'Возврат инициирован',
  };
  return map[s] || s || '—';
};

const ruOutboxStatus = (s) => {
  const map = {
    new: 'новое',
    processing: 'в обработке',
    processed: 'отправлено',
  };
  return map[s] || s || '—';
};

const ruOrderStatus = (s) => {
  const map = {
    CREATED: 'Создан',
    PAYMENT_AUTHORIZED: 'Оплата подтверждена',
    TICKET_ISSUED: 'Билет оформлен',
    CANCELLED: 'Отменен',
    REFUND_PENDING: 'Возврат в обработке',
  };
  return map[s] || s || '—';
};

const ruTicketStatus = (s) => {
  const map = {
    ISSUED: 'Выпущен',
  };
  return map[s] || s || '—';
};

const KV = ({ k, v, raw }) => (
  <div className="wf-kv-row">
    <div className="wf-kv-key">{k}</div>
    <div className="wf-kv-val">
      <span className="wf-mono">{v ?? '—'}</span>
      {raw ? <span className="wf-kv-raw">{raw}</span> : null}
    </div>
  </div>
);

const Badge = ({ children, tone = 'neutral' }) => (
  <span className={`wf-badge wf-badge--${tone}`}>{children}</span>
);

const Step = ({ title, status, badges, why, details }) => (
  <div className={`wf-step wf-step--${status}`}>
    <div className="wf-step-head">
      <div className="wf-step-title">
        <span className="wf-dot" />
        {title}
      </div>
      <div className="wf-step-badges">{badges}</div>
    </div>
    {why && <div className="wf-step-why">{why}</div>}
    {details && <div className="wf-step-details">{details}</div>}
  </div>
);

const findOutbox = (workflow, type) => (workflow?.outbox || []).find((e) => e.event_type === type);
const findInbox = (workflow, consumer, type) => (workflow?.inbox || []).find((e) => e.consumer === consumer && e.event_type === type);

export default function Workflow({ orderId }) {
  const [workflow, setWorkflow] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (!orderId) return;

    let cancelled = false;
    const tick = async () => {
      try {
        const res = await fetch(`/api/orders/${orderId}/workflow`, { cache: 'no-store' });
        if (!res.ok) throw new Error(`workflow fetch failed (${res.status})`);
        const data = await res.json();
        if (!cancelled) {
          setWorkflow(data);
          setError(null);
        }
      } catch (e) {
        if (!cancelled) setError(e.message || 'workflow fetch failed');
      }
    };

    tick();
    const id = setInterval(tick, 800);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [orderId]);

  const steps = useMemo(() => {
    if (!workflow?.order) return [];

    const outOrderCreated = findOutbox(workflow, 'OrderCreated');
    const inPaymentOrderCreated = findInbox(workflow, 'payment-service', 'OrderCreated');
    const outPaymentAuthorized = findOutbox(workflow, 'PaymentAuthorized');
    const inTicketPaymentAuthorized = findInbox(workflow, 'ticket-service', 'PaymentAuthorized');
    const outTicketIssued = findOutbox(workflow, 'TicketIssued');
    const inOrderTicketIssued = findInbox(workflow, 'order-service', 'TicketIssued');

    const published = (e) => e && e.status === 'processed';

    const order = workflow.order;

    return [
      {
        title: 'API: одна транзакция в Postgres (orders + outbox)'
          + (outOrderCreated?.created_at ? ` • ${fmtTime(outOrderCreated.created_at)}` : ''),
        status: outOrderCreated ? 'done' : 'pending',
        badges: [<Badge key="outbox" tone="accent">OUTBOX</Badge>],
        why: 'Transactional Outbox: запись в orders и запись в outbox фиксируются атомарно в одной транзакции.',
        details: outOrderCreated ? (
          <>
            <div className="wf-block-title">Что записали в `orders`</div>
            <div className="wf-kv">
              <KV k="id" v={order.id} />
              <KV k="user_id" v={order.user_id} />
              <KV k="status" v={ruOrderStatus(order.status)} raw={order.status} />
              <KV k="total_amount" v={order.total_amount} />
              <KV k="from_city" v={order.from_city} />
              <KV k="to_city" v={order.to_city} />
              <KV k="travel_date" v={order.travel_date} />
              <KV k="travel_time" v={order.travel_time} />
              <KV k="airline" v={order.airline} />
              <KV k="created_at" v={order.created_at} />
            </div>

            <div className="wf-block-title" style={{ marginTop: 10 }}>Что записали в `outbox` (в той же транзакции)</div>
            <div className="wf-kv">
              <KV k="id" v={outOrderCreated.id} />
              <KV k="event_type" v={ruEventType(outOrderCreated.event_type)} raw={outOrderCreated.event_type} />
              <KV k="correlation_id" v={outOrderCreated.correlation_id || order.id} />
              <KV k="producer" v={ruService(outOrderCreated.producer)} raw={outOrderCreated.producer} />
              <KV k="status" v={ruOutboxStatus(outOrderCreated.status)} raw={outOrderCreated.status} />
            </div>

            <div className="wf-note">
              `payload` в outbox содержит JSON заказа (в проекте мы показываем его через поля `orders`, чтобы не грузить новичка JSON-ом).
            </div>
          </>
        ) : (
          <>Ждем запись в `outbox` (OrderCreated).</>
        ),
      },
      {
        title: 'Воркер (Outbox Poller): забрал записи из outbox и опубликовал в Kafka'
          + (outOrderCreated?.updated_at ? ` • ${fmtTime(outOrderCreated.updated_at)}` : ''),
        status: published(outOrderCreated) ? 'done' : outOrderCreated ? 'active' : 'pending',
        badges: [<Badge key="outbox" tone="accent">OUTBOX</Badge>],
        why: 'Воркер отдельно от API читает outbox и публикует события. Так API не зависит от Kafka и не теряет события при сбоях.',
        details: outOrderCreated ? (
          <>
            <div className="wf-block-title">Что происходит</div>
            <div className="wf-note">
              Воркер делает `FetchBatch` (claim через `FOR UPDATE SKIP LOCKED`), помечает запись `processing`,
              публикует в Kafka и затем ставит `processed`.
            </div>
            <div className="wf-kv" style={{ marginTop: 10 }}>
              <KV k="outbox.status" v={ruOutboxStatus(outOrderCreated.status)} raw={outOrderCreated.status} />
              <KV k="kafka.topic" v="orders-events" />
              <KV k="kafka.key" v={outOrderCreated.correlation_id || outOrderCreated.id} />
              <KV k="сообщение" v="envelope: {id, type, correlation_id, producer, occurred_at, payload}" />
            </div>
          </>
        ) : null,
      },
      {
        title: `${ruService('payment-service')}: Inbox (дедуп) + локальная транзакция оплаты`
          + (inPaymentOrderCreated?.processed_at ? ` • ${fmtTime(inPaymentOrderCreated.processed_at)}` : ''),
        status: inPaymentOrderCreated ? 'done' : published(outOrderCreated) ? 'active' : 'pending',
        badges: [
          <Badge key="inbox" tone="good">INBOX</Badge>,
          <Badge key="saga" tone="neutral">SAGA</Badge>,
        ],
        why: 'Inbox (таблица inbox_events) защищает от дублей Kafka. Saga: сервис реагирует на событие и публикует следующее.',
        details: inPaymentOrderCreated ? (
          <>
            <div className="wf-block-title">Что записали в `inbox_events`</div>
            <div className="wf-kv">
              <KV k="consumer" v={ruService(inPaymentOrderCreated.consumer)} raw={inPaymentOrderCreated.consumer} />
              <KV k="event_id" v={inPaymentOrderCreated.event_id} />
              <KV k="event_type" v={ruEventType(inPaymentOrderCreated.event_type)} raw={inPaymentOrderCreated.event_type} />
              <KV k="correlation_id" v={inPaymentOrderCreated.correlation_id} />
              <KV k="processed_at" v={inPaymentOrderCreated.processed_at} />
            </div>

            <div className="wf-block-title" style={{ marginTop: 10 }}>Что записали в `payments`</div>
            {workflow.payment ? (
              <div className="wf-kv">
                <KV k="id" v={workflow.payment.id} />
                <KV k="order_id" v={workflow.payment.order_id} />
                <KV k="status" v={workflow.payment.status} />
                <KV k="amount" v={workflow.payment.amount} />
              </div>
            ) : (
              <div className="wf-note">Ждем запись в `payments`...</div>
            )}
          </>
        ) : (
          <>Ждем обработку OrderCreated у payment-service.</>
        ),
      },
      {
        title: `${ruService('payment-service')}: записал событие PaymentAuthorized в outbox`
          + (outPaymentAuthorized?.created_at ? ` • ${fmtTime(outPaymentAuthorized.created_at)}` : ''),
        status: outPaymentAuthorized ? 'done' : inPaymentOrderCreated ? 'active' : 'pending',
        badges: [<Badge key="outbox" tone="accent">OUTBOX</Badge>, <Badge key="saga" tone="neutral">SAGA</Badge>],
        why: 'Saga choreography: следующее действие запускается событием, без централизованного оркестратора.',
        details: outPaymentAuthorized ? (
          <>
            <div className="wf-kv">
              <KV k="event_type" v={ruEventType(outPaymentAuthorized.event_type)} raw={outPaymentAuthorized.event_type} />
              <KV k="correlation_id" v={outPaymentAuthorized.correlation_id} />
              <KV k="causation_id" v={outPaymentAuthorized.causation_id || '—'} />
              <KV k="producer" v={ruService(outPaymentAuthorized.producer)} raw={outPaymentAuthorized.producer} />
              <KV k="status" v={ruOutboxStatus(outPaymentAuthorized.status)} raw={outPaymentAuthorized.status} />
            </div>
          </>
        ) : null,
      },
      {
        title: `${ruService('ticket-service')}: Inbox (дедуп) + выпуск билета`
          + (inTicketPaymentAuthorized?.processed_at ? ` • ${fmtTime(inTicketPaymentAuthorized.processed_at)}` : ''),
        status: inTicketPaymentAuthorized ? 'done' : published(outPaymentAuthorized) ? 'active' : 'pending',
        badges: [
          <Badge key="inbox" tone="good">INBOX</Badge>,
          <Badge key="saga" tone="neutral">SAGA</Badge>,
        ],
        why: 'Ticket Service автономно выполняет свой локальный шаг и публикует TicketIssued.',
        details: workflow.ticket ? (
          <>
            <div className="wf-block-title">Что записали в `tickets`</div>
            <div className="wf-kv">
              <KV k="id" v={workflow.ticket.id} />
              <KV k="order_id" v={workflow.ticket.order_id} />
              <KV k="from_city" v={workflow.ticket.from_city || workflow.order.from_city} />
              <KV k="to_city" v={workflow.ticket.to_city || workflow.order.to_city} />
              <KV k="travel_date" v={workflow.ticket.travel_date || workflow.order.travel_date} />
              <KV k="travel_time" v={workflow.ticket.travel_time || workflow.order.travel_time} />
              <KV k="airline" v={workflow.ticket.airline || workflow.order.airline} />
              <KV k="status" v={ruTicketStatus(workflow.ticket.status)} raw={workflow.ticket.status} />
            </div>
          </>
        ) : (
          <>Ждем выпуск билета и запись в `tickets`.</>
        ),
      },
      {
        title: `${ruService('ticket-service')}: записал событие TicketIssued в outbox`
          + (outTicketIssued?.created_at ? ` • ${fmtTime(outTicketIssued.created_at)}` : ''),
        status: outTicketIssued ? 'done' : inTicketPaymentAuthorized ? 'active' : 'pending',
        badges: [<Badge key="outbox" tone="accent">OUTBOX</Badge>, <Badge key="saga" tone="neutral">SAGA</Badge>],
        why: 'Событие TicketIssued закрывает сагу: далее “order-service” отражает финальное состояние заказа.',
        details: outTicketIssued ? (
          <>
            <div className="wf-kv">
              <KV k="event_type" v={ruEventType(outTicketIssued.event_type)} raw={outTicketIssued.event_type} />
              <KV k="correlation_id" v={outTicketIssued.correlation_id} />
              <KV k="causation_id" v={outTicketIssued.causation_id || '—'} />
              <KV k="producer" v={ruService(outTicketIssued.producer)} raw={outTicketIssued.producer} />
              <KV k="status" v={ruOutboxStatus(outTicketIssued.status)} raw={outTicketIssued.status} />
            </div>
          </>
        ) : null,
      },
      {
        title: `${ruService('order-service')}: Inbox (дедуп) + финальный статус заказа`
          + (inOrderTicketIssued?.processed_at ? ` • ${fmtTime(inOrderTicketIssued.processed_at)}` : ''),
        status: workflow.order.status === 'TICKET_ISSUED' ? 'done' : outTicketIssued ? 'active' : 'pending',
        badges: [<Badge key="inbox" tone="good">INBOX</Badge>, <Badge key="saga" tone="neutral">SAGA</Badge>],
        why: 'Заказ становится “готов” только после обработки TicketIssued (без 2PC).',
        details: (
          <>Текущий статус заказа: <span className="wf-mono">{ruOrderStatus(workflow.order.status)}</span> <span className="wf-mono">({workflow.order.status})</span>.</>
        ),
      },
    ];
  }, [workflow]);

  return (
    <div className="card wf">
      <div className="wf-header">
        <div>
          <div className="wf-title">Воркфлоу: Outbox + Inbox + Saga (хореография)</div>
          <div className="wf-sub">
            Order ID: <span className="wf-mono">{orderId}</span>
          </div>
        </div>
        {workflow?.order && (
          <div className="wf-status">
            <div className="wf-status-label">Статус заказа</div>
            <div className="wf-status-value">{ruOrderStatus(workflow.order.status)}</div>
            <div className="wf-status-raw">{workflow.order.status}</div>
          </div>
        )}
      </div>

      {error && <div className="wf-error">{error}</div>}

      <div className="wf-steps">
        {steps.length === 0 ? (
          <div className="wf-empty">Ждем данные workflow...</div>
        ) : (
          steps.map((s, idx) => (
            <Step
              key={idx}
              title={s.title}
              status={s.status}
              badges={s.badges}
              why={s.why}
              details={s.details}
            />
          ))
        )}
      </div>

      {workflow?.ticket && (
        <div className="wf-result">
          <div className="wf-result-title">Готовый билет</div>
          <div className="wf-result-card">
            <div className="wf-result-row">
              <span className="wf-mono">{workflow.ticket.from_city || workflow.order.from_city || '—'}</span>
              <span className="wf-arrow">→</span>
              <span className="wf-mono">{workflow.ticket.to_city || workflow.order.to_city || '—'}</span>
            </div>
            <div className="wf-result-meta">
              {workflow.ticket.travel_date || workflow.order.travel_date || ''} {workflow.ticket.travel_time || workflow.order.travel_time || ''}
              {workflow.ticket.airline || workflow.order.airline ? ` • ${workflow.ticket.airline || workflow.order.airline}` : ''}
              {workflow.ticket.status ? ` • ${ruTicketStatus(workflow.ticket.status)} (${workflow.ticket.status})` : ''}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
