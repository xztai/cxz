import { redirect } from '@sveltejs/kit';

// Old bookmarks lead to login; resending is available in the pending-email card.
export const GET = () => redirect(303, '/auth/login');
