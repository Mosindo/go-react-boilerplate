export type AuthStackParamList = {
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  ResetPassword: undefined;
};

export type MainTabParamList = {
  Discover: undefined;
  Matches: undefined;
  Chats: undefined;
  Me: undefined;
};

export type MainStackParamList = {
  Tabs: undefined;
  Thread: { conversationId: string; userId: string; name: string };
  ProfileDetail: { userId: string; canAct?: boolean };
  Report: { userId: string; name: string };
  Notifications: undefined;
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Blocked: undefined;
};
