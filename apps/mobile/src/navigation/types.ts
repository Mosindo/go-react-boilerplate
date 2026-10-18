export type AuthStackParamList = {
  Auth: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Home: undefined;
  Messages: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type ConversationParams = {
  conversationId: string;
  matchId?: string;
  userId: string;
  name?: string;
};

export type MainStackParamList = {
  Tabs: undefined;
  Conversation: ConversationParams;
  Person: { userId: string; matchId?: string; conversationId?: string };
  EditProfile: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
};
