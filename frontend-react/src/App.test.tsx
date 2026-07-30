import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import App from './App';

// Mock do WebSocket
jest.mock('socket.io-client', () => {
  const emit = jest.fn();
  const on = jest.fn();
  const connect = jest.fn();
  return jest.fn(() => ({ emit, on, connect }));
});

test('renders app and shows login', () => {
  render(<App />);
  expect(screen.getByText(/Atendimentos/i)).toBeInTheDocument();
});

test('can send message', async () => {
  render(<App />);
  
  // Selecionar cliente
  fireEvent.click(screen.getByText(/João Silva/i));
  
  // Digitar mensagem
  const input = screen.getByPlaceholderText(/Digite sua mensagem/i);
  fireEvent.change(input, { target: { value: 'Test message' } });
  
  // Enviar
  fireEvent.click(screen.getByText(/Enviar/i));
  
  await waitFor(() => {
    expect(screen.getByText(/Test message/i)).toBeInTheDocument();
  });
});