import type { Card } from "../api/types";

export type AuthStackParams = {
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
};

export type TabParams = {
  Discover: undefined;
  Likes: undefined;
  Messages: undefined;
  Me: undefined;
};

export type MainStackParams = {
  Tabs: undefined;
  Chat: { conversationId: string; matchId: string; user: Card };
  UserProfile: { user: Card; conversationId?: string; matchId?: string };
  EditProfile: undefined;
  Settings: undefined;
  Notifications: undefined;
  Blocked: undefined;
};
