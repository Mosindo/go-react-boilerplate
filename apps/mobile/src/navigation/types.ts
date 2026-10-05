import type { NativeStackScreenProps } from "@react-navigation/native-stack";

export type RootStackParamList = {
  Auth: undefined;
  ForgotPassword: undefined;
  ResetPassword: { token?: string } | undefined;
  Onboarding: undefined;
  Tabs: undefined;
  ProfileDetail: { userId: string; matchId?: string; conversationId?: string };
  Chat: { conversationId: string; matchId: string; userId: string; name: string; photoPath?: string };
  Notifications: undefined;
  EditProfile: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Me: undefined;
};

export type RootScreenProps<T extends keyof RootStackParamList> = NativeStackScreenProps<RootStackParamList, T>;
