import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps, NavigatorScreenParams } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Activity: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
  Onboarding: undefined;
  CompleteProfile: undefined;
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { userId: string; firstName?: string; photoUrl?: string };
  ProfileDetail: { userId: string; fromChat?: boolean };
  MyPreview: undefined;
  EditProfile: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
  Report: { userId: string; firstName: string };
};

export type RootScreenProps<T extends keyof RootStackParamList> = NativeStackScreenProps<RootStackParamList, T>;

export type TabScreenProps<T extends keyof TabParamList> = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, T>,
  NativeStackScreenProps<RootStackParamList>
>;
