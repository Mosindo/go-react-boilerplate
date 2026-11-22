import type { NavigatorScreenParams } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

export type AuthStackParamList = {
  Welcome: undefined;
  SignIn: undefined;
  SignUp: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Discover: undefined;
  Likes: undefined;
  Messages: undefined;
  Me: undefined;
};

export type ProfileContext = "discover" | "likes" | "match";

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Onboarding: undefined;
  ChatThread: { userId: string; firstName: string; photoUrl?: string };
  ProfileDetail: { userId: string; context: ProfileContext };
  EditProfile: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
  Notifications: undefined;
};

export type AuthScreenProps<T extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, T>;
export type MainScreenProps<T extends keyof MainStackParamList> = NativeStackScreenProps<MainStackParamList, T>;
