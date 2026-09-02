import { ReserveForm } from '@/components/ReserveForm';

export default function Home() {
  return (
    <main className="min-h-screen bg-gray-50 py-12 px-4">
      <div className="max-w-3xl mx-auto">
        <h1 className="text-4xl font-bold text-center text-gray-800 mb-2">
          Inventory Management
        </h1>
        <p className="text-center text-gray-500 mb-8">
          Reserve products with idempotency and real-time processing
        </p>
        <ReserveForm />
      </div>
    </main>
  );
}