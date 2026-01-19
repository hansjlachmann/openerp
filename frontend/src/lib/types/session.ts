export interface SessionState {
	database: string | null;
	company: string | null;
	userID: string | null;
	userName: string | null;
	userFullName: string | null;
	language: string;
	isAuthenticated: boolean;
	isLoading: boolean;
}

export interface SessionResponse {
	database: string;
	company: string;
	user_id: string;
	user_name: string;
	user_full_name: string;
	language: string;
}
