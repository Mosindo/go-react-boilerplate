import type { NavigatorScreenParams } from "@react-navigation/native";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type MainTabParamList = {
  Discover: undefined;
  Likes: undefined;
  Chats: undefined;
  Me: undefined;
};

export type AppStackParamList = {
  Tabs: NavigatorScreenParams<MainTabParamList>;
  ProfileDetail: { userId: string; canSwipe?: boolean };
  Chat: { conversationId: string; name?: string };
  Notifications: undefined;
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
  DeleteAccount: undefined;
};
