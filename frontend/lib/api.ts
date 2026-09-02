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

export async function reserveProduct(data: ReserveRequest): Promise<ReserveResponse> {
    const res = await fetch('/api/v1/reserve', {
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