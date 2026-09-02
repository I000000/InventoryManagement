'use client';

import { useEffect, useState } from 'react';
import { getReservations, ReserveLogEntry } from '@/lib/api';

interface ReservationListProps {
  refreshTrigger?: number;
}

export function ReservationList({ refreshTrigger = 0 }: ReservationListProps) {
  const [reservations, setReservations] = useState<ReserveLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const data = await getReservations(10);
        setReservations(data);
        setError(null);
      } catch (err) {
        setError('Failed to load reservations');
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [refreshTrigger]);

  if (loading) {
    return (
      <div className="mt-8">
        <h2 className="text-xl font-semibold text-gray-700 mb-4">Recent Reservations</h2>
        <p className="text-gray-500">Loading...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="mt-8">
        <h2 className="text-xl font-semibold text-gray-700 mb-4">Recent Reservations</h2>
        <p className="text-red-500">{error}</p>
      </div>
    );
  }

  return (
    <div className="mt-8">
      <h2 className="text-xl font-semibold text-gray-700 mb-4">Recent Reservations</h2>
      {!reservations || reservations.length === 0 ? (
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
                {new Date(entry.created_at).toLocaleString()}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}