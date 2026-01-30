import React, { useState } from 'react';

const CITIES = [
  "Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", "Казань",
  "Нижний Новгород", "Челябинск", "Самара", "Омск", "Ростов-на-Дону",
  "Уфа", "Красноярск", "Воронеж", "Пермь", "Волгоград",
  "Краснодар", "Саратов", "Тюмень", "Тольятти", "Ижевск"
];

const SearchForm = ({ onSearch }) => {
  const [from, setFrom] = useState('Москва');
  const [to, setTo] = useState('Санкт-Петербург');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);

  const handleSubmit = (e) => {
    e.preventDefault();
    onSearch({ from, to, date });
  };

  return (
    <form onSubmit={handleSubmit} className="card">
      <div className="input-group">
        <select value={from} onChange={(e) => setFrom(e.target.value)}>
          {CITIES.map(city => <option key={city} value={city}>{city}</option>)}
        </select>
        
        <select value={to} onChange={(e) => setTo(e.target.value)}>
          {CITIES.map(city => <option key={city} value={city}>{city}</option>)}
        </select>
        
        <input 
          type="date" 
          value={date} 
          onChange={(e) => setDate(e.target.value)} 
        />
      </div>
      
      <button type="submit" className="btn-primary">
        Найти билеты
      </button>
    </form>
  );
};

export default SearchForm;
