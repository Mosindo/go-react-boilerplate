import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  Auth: undefined;
  ForgotPassword: { email?: string } | undefined;
  Onboarding: undefined;
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Thread: { conversationId: string; peerId: string; peerName: string };
  ProfileDetail: { userId: string; conversationId?: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  Blocked: undefined;
  Security: undefined;
  Report: { userId: string; name: string };
};
