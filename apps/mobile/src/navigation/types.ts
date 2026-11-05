export type AuthStackParams = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  Forgot: undefined;
  Reset: { email?: string } | undefined;
};

export type MainStackParams = {
  Tabs: undefined;
  Chat: { conversationId: string; peerId: string; peerName: string; peerPhotoId: string | null };
  UserProfile: { userId: string; canSwipe?: boolean };
  EditProfile: undefined;
  Preferences: undefined;
  Settings: undefined;
  Blocked: undefined;
};

export type TabParams = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Me: undefined;
};
