import type { Card } from "../api/types";

export type AuthStackParams = {
  Auth: undefined;
  ForgotPassword: undefined;
};

export type MainStackParams = {
  Tabs: undefined;
  ProfileDetail: { userId: string; card?: Card; canAct?: boolean };
  Chat: { conversationId: string; peerName: string; peerId: string };
  EditProfile: undefined;
  Settings: undefined;
  Blocked: undefined;
  Notifications: undefined;
};

export type TabParams = {
  Discover: undefined;
  Likes: undefined;
  Messages: undefined;
  Me: undefined;
};
