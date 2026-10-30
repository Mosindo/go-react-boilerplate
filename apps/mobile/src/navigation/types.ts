export type AuthStackParamList = {
  Auth: undefined;
  Forgot: undefined;
};

export type MainStackParamList = {
  Tabs: undefined;
  Chat: { conversationId: string; userId: string; name: string };
  ProfileDetail: { userId: string; showActions?: boolean };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Settings: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Me: undefined;
};
