import type { CompositeScreenProps, NavigatorScreenParams } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: { notice?: string } | undefined;
  Register: undefined;
  ForgotPassword: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type AppStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList>;
  Chat: { matchId: string };
  ProfileDetail: { userId: string; fromChat?: boolean };
  EditProfile: undefined;
  Preferences: undefined;
  Privacy: undefined;
  BlockedUsers: undefined;
  DeleteAccount: undefined;
};

export type AuthScreenProps<T extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, T>;
export type AppScreenProps<T extends keyof AppStackParamList> = NativeStackScreenProps<AppStackParamList, T>;
export type TabScreenProps<T extends keyof TabParamList> = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, T>,
  NativeStackScreenProps<AppStackParamList>
>;
