import React from 'react';

const TicketList = ({ tickets, onBuy }) => {
    if (!tickets || tickets.length === 0) return null;

    return (
        <div className="card">
            {tickets.map((ticket) => (
                <div key={ticket.id} className="ticket">
                    <div className="ticket-info">
                        <div className="ticket-route">
                            {ticket.from} → {ticket.to}
                        </div>
                        <div className="ticket-meta">
                            {ticket.date} • {ticket.time} • {ticket.airline}
                        </div>
                    </div>

                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '10px' }}>
                        <div className="price">{ticket.price} ₽</div>
                        <button
                            className="btn-secondary"
                            onClick={() => onBuy(ticket)}
                        >
                            Купить
                        </button>
                    </div>
                </div>
            ))}
        </div>
    );
};

export default TicketList;
