'use client';

import { useState } from 'react';
import { reserveProduct } from '@/lib/api';

const PRODUCTS = [
  'iphone_15_pro',
  'iphone_15',
  'samsung_s24_ultra',
  'samsung_s24',
  'sony_wh1000xm5',
  'apple_watch_series9',
  'macbook_air_m2',
  'macbook_pro_m3',
  'ipad_pro_12_9',
  'airpods_pro_2',
];

interface ReservationFormProps {
  onSuccess?: () => void;
}

export function ReservationForm({ onSuccess }: ReservationFormProps) {
  const [productId, setProductId] = useState(PRODUCTS[0]);
  const [quantity, setQuantity] = useState(1);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setResult(null);

    const requestId = `web-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;

    try {
      const data = await reserveProduct({ product_id: productId, quantity, request_id: requestId });
      setResult({
        success: true,
        message: `✅ Reserved ${data.reserved} x ${data.product_id}`,
      });
      if (onSuccess) onSuccess();
    } catch (err: any) {
      setResult({
        success: false,
        message: `❌ Error: ${err.message || 'Something went wrong'}`,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-md p-6">
      <form onSubmit={handleSubmit} className="space-y-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Product</label>
          <select
            value={productId}
            onChange={(e) => setProductId(e.target.value)}
            className="w-full border border-gray-300 rounded-lg px-4 py-2 bg-white text-gray-800 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            {PRODUCTS.map((p) => (
              <option key={p} value={p}>{p}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Quantity</label>
          <input
            type="number"
            min={1}
            max={100}
            value={quantity}
            onChange={(e) => setQuantity(Number(e.target.value))}
            className="w-full border border-gray-300 rounded-lg px-4 py-2 bg-white text-gray-800 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white py-3 rounded-lg font-medium hover:bg-blue-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? 'Reserving...' : 'Reserve'}
        </button>
      </form>

      {result && (
        <div
          className={`mt-4 p-4 rounded-lg ${
            result.success ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'
          }`}
        >
          {result.message}
        </div>
      )}
    </div>
  );
}