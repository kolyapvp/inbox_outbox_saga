import React, { useEffect } from 'react';

const Alert = ({ title, message, onClose }) => {
    useEffect(() => {
        const timer = setTimeout(() => {
            if (onClose) onClose();
        }, 5000); // Auto hide after 5s if desired, or keep until user dismisses
        return () => clearTimeout(timer);
    }, [onClose]);

    return (
        <div className="alert-toast">
            <div style={{ fontSize: '24px' }}>✅</div>
            <div>
                <div className="alert-title">{title}</div>
                <div className="alert-desc">{message}</div>
            </div>
        </div>
    );
};

export default Alert;
