import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { conversationId: string; peerId: string; name: string };
  ProfileDetail: { userId: string; conversationId?: string; matchId?: string };
  EditProfile: undefined;
  Preferences: undefined;
  BlockedUsers: undefined;
};

export type AuthStackParamList = {
  Auth: undefined;
  Forgot: undefined;
};
