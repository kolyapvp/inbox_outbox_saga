import React, { useRef, useState, useEffect } from 'react';
import SearchForm from './components/SearchForm';
import TicketList from './components/TicketList';
import Alert from './components/Alert';
import Workflow from './components/Workflow';

// Mock ticket generation
const generateTickets = (from, to, date) => {
  const tickets = [];
  const times = ['07:30', '10:15', '14:45', '18:20', '21:50'];
  const airlines = ['Aeroflot', 'S7 Airlines', 'Pobeda', 'Ural Airlines', 'Utair'];
  const basePrice = 4500;

  for (let i = 0; i < 5; i++) {
    tickets.push({
      id: Math.random().toString(36).substr(2, 9),
      from,
      to,
      date,
      time: times[i],
      airline: airlines[i % airlines.length],
      price: basePrice + Math.floor(Math.random() * 5000)
    });
  }
  return tickets;
};

function App() {
  const [tickets, setTickets] = useState([]);
  const [activeOrder, setActiveOrder] = useState(null); // { order_id, done }
  const [alert, setAlert] = useState(null);
  const [loading, setLoading] = useState(false);

  const workflowRef = useRef(null);

  const handleSearch = (searchParams) => {
    // Reset ticket list with new search
    setTickets(generateTickets(searchParams.from, searchParams.to, searchParams.date));
  };

  const handleBuy = async (ticket) => {
    setLoading(true);
    try {
      // API call to create order
      const response = await fetch('/api/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', // Mock User ID from seed
          amount: ticket.price,
          from: ticket.from,
          to: ticket.to,
          date: ticket.date,
          time: ticket.time,
          airline: ticket.airline,
        })
      });

      if (!response.ok) {
        throw new Error('Failed to create order');
      }

      const data = await response.json();
      console.log('Order created:', data);

      // Look for order_id in response. 
      // Based on our changes it should be { status: "created", order_id: "uuid" }
      if (data.order_id) {
        setActiveOrder({ order_id: data.order_id, done: false });
        setAlert({
          title: 'Заказ создан',
          message: 'Ваш билет оформляется. Пожалуйста, подождите...'
        });

        // Scroll to the human-readable workflow section
        requestAnimationFrame(() => {
          workflowRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
        });
      } else {
        setAlert({ title: 'Заказ создан', message: 'Обработка...' });
      }

    } catch (err) {
      console.error(err);
      setAlert({ title: 'Ошибка', message: 'Не удалось оформить билет.' });
    } finally {
      setLoading(false);
    }
  };

  // Polling Effect
  useEffect(() => {
    if (!activeOrder?.order_id || activeOrder.done) return;

    let pollInterval;
    const pollStatus = async () => {
      try {
        const res = await fetch(`/api/orders/${activeOrder.order_id}`);
        if (!res.ok) return;
        const order = await res.json();

        console.log('Polling order:', order);

        if (order.status === 'TICKET_ISSUED') {
          setAlert({
            title: 'Билет оформлен!',
            message: `Ваш билет до ${tickets[0]?.to || 'пункта назначения'} успешно забронирован. Проверьте вашу почту.`
          });
          setActiveOrder((prev) => ({ ...prev, done: true }));
        }

        if (order.status === 'CANCELLED') {
          setAlert({
            title: 'Заказ отменен',
            message: 'Оплата не прошла или заказ был отменен. Попробуйте еще раз.'
          });
          setActiveOrder((prev) => ({ ...prev, done: true }));
        }
      } catch (e) {
        console.error('Polling error', e);
      }
    };

    // Poll every 1 second
    pollInterval = setInterval(pollStatus, 1000);

    return () => clearInterval(pollInterval);
  }, [activeOrder, tickets]);

  return (
    <div className="container">
      {alert && (
        <Alert
          title={alert.title}
          message={alert.message}
          onClose={() => setAlert(null)}
        />
      )}

      <h1>Aviasales Clone</h1>

      <SearchForm onSearch={handleSearch} />

      <TicketList tickets={tickets} onBuy={handleBuy} />

      <div ref={workflowRef} />
      {activeOrder?.order_id && <Workflow orderId={activeOrder.order_id} />}

      {loading && (
        <div style={{ textAlign: 'center', marginTop: '20px', color: '#8b9bb4' }}>
          Processing transaction...
        </div>
      )}
    </div>
  );
}

export default App;
