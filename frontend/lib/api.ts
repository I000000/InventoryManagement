const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface ReserveRequest {
  product_id: string;
  quantity: number;
  request_id: string;
}

export interface ReserveResponse {
  status: string;
  product_id: string;
  reserved: number;
}

export interface ReserveLogEntry {
    id: number;
    product_id: string;
    quantity: number;
    request_id: string;
    user_id: string;
    status: string;
    error_message: string | null;
    created_at: string;
  }

export async function reserveProduct(data: ReserveRequest): Promise<ReserveResponse> {
    const res = await fetch('/api/v1/reservations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  
    if (!res.ok) {
      const error = await res.json().catch(() => ({}));
      throw new Error(error.error || `HTTP ${res.status}`);
    }
  
    return res.json();
  }

  export async function getReservations(limit = 20): Promise<ReserveLogEntry[]> {
    const res = await fetch(`/api/v1/reservations?limit=${limit}`);
    if (!res.ok) {
      throw new Error('Failed to fetch reservations');
    }
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  }