import type { NavigatorScreenParams } from '@react-navigation/native';
import type { ProfileSummary } from '../api/types';

export type ChatUser = Pick<ProfileSummary, 'id' | 'firstName' | 'mainPhotoUrl'>;

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  ResetPassword: { email: string };
};

export type TabParamList = {
  Discover: undefined;
  Likes: undefined;
  Messages: undefined;
  Profile: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  ProfileDetail: {
    userId: string;
    mode?: 'swipe' | 'view';
    summary?: Pick<ProfileSummary, 'firstName'>;
  };
  Chat: { conversationId: string; user: ChatUser };
  MyPreview: undefined;
  EditProfile: undefined;
  EditPhotos: undefined;
  EditPreferences: undefined;
  Privacy: undefined;
  Notifications: undefined;
  Blocked: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
};
