export type AuthStackParams = {
  Login: undefined;
  Register: undefined;
  Forgot: undefined;
};

export type MainStackParams = {
  Tabs: undefined;
  Chat: { conversationId: string; name: string; userId: string };
  ProfileDetail: { userId: string; conversationId?: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Blocked: undefined;
  ChangePassword: undefined;
};

export type TabParams = {
  Discover: undefined;
  Likes: undefined;
  Messages: undefined;
  Notifications: undefined;
  Me: undefined;
};
