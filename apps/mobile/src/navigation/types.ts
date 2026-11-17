import type { CompositeScreenProps, NavigatorScreenParams } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";

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
  ResetPassword: { email: string };
  Onboarding: undefined;
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Conversation: { conversationId: string };
  ProfileDetail: { userId: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Blocked: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
  Report: { userId: string; firstName: string };
};

export type RootScreenProps<T extends keyof RootStackParamList> = NativeStackScreenProps<RootStackParamList, T>;

export type TabScreenProps<T extends keyof TabParamList> = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, T>,
  NativeStackScreenProps<RootStackParamList>
>;

declare global {
  namespace ReactNavigation {
    // eslint-disable-next-line @typescript-eslint/no-empty-object-type
    interface RootParamList extends RootStackParamList {}
  }
}
