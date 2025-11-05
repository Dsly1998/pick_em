import { createClient } from '@supabase/supabase-js';
import { env } from '$env/dynamic/public';

const url = env.PUBLIC_SUPABASE_URL;
const anonKey = env.PUBLIC_SUPABASE_ANON_KEY;

if (!url || !anonKey) {
	throw new Error('Missing Supabase env vars: PUBLIC_SUPABASE_URL and PUBLIC_SUPABASE_ANON_KEY');
}

// Export a single client instance to reuse across the app.
export const supabase = createClient(url, anonKey, {
	auth: {
		persistSession: true
	}
});
