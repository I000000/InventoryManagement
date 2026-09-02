'use client';

import { useEffect, useState } from 'react';
import { getReservations, ReserveLogEntry } from '@/lib/api';

interface ReservationListProps {
  refreshTrigger?: number;
}

export function ReservationList({ refreshTrigger = 0 }: ReservationListProps) {
  const [reservations, setReservations] = useState<ReserveLogEntry[]>([]);
  const [loading, setLoading] = useState(true);

  // Загрузка начальных данных (при монтировании и при изменении refreshTrigger)
  const fetchReservations = async () => {
    try {
      const data = await getReservations(10);
      setReservations(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch reservations', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchReservations();
  }, [refreshTrigger]);

  // WebSocket — слушаем новые резервирования
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'new_reservation') {
          const payload = data.payload;
          const newEntry: ReserveLogEntry = {
            id: Date.now(), // временный ID
            product_id: payload.product_id,
            quantity: payload.quantity,
            status: payload.status,
            created_at: payload.created_at,
            request_id: '',
            user_id: '',
            error_message: null,
          };
          setReservations((prev) => [newEntry, ...prev]);
        }
      } catch (err) {
        console.error('WebSocket message parse error', err);
      }
    };

    ws.onerror = (err) => {
      console.error('WebSocket error', err);
    };

    return () => {
      ws.close();
    };
  }, []);

  if (loading) {
    return (
      <div className="mt-8">
        <h2 className="text-xl font-semibold text-gray-700 mb-4">Recent Reservations</h2>
        <p className="text-gray-500">Loading...</p>
      </div>
    );
  }

  return (
    <div className="mt-8">
      <h2 className="text-xl font-semibold text-gray-700 mb-4">Recent Reservations</h2>
      {reservations.length === 0 ? (
        <p className="text-gray-500">No reservations yet.</p>
      ) : (
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {reservations.map((entry) => (
            <div key={entry.id} className="bg-gray-50 p-3 rounded-lg text-sm flex justify-between items-center">
              <div>
                <span className="font-medium text-gray-800">{entry.product_id}</span>
                <span className="ml-2 text-gray-600">×{entry.quantity}</span>
                <span className={`ml-4 px-2 py-0.5 rounded-full text-xs font-medium ${
                  entry.status === 'success' ? 'bg-green-100 text-green-800' :
                  entry.status === 'duplicate' ? 'bg-yellow-100 text-yellow-800' :
                  'bg-red-100 text-red-800'
                }`}>
                  {entry.status}
                </span>
              </div>
              <span className="text-gray-400 text-xs">
                {entry.created_at ? new Date(entry.created_at).toLocaleString() : 'N/A'}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}