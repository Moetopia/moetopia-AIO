import type { paths } from './schema'

export type { paths, components, operations } from './schema'
export type StatusResponse = paths['/api/v1/status']['get']['responses'][200]['content']['application/json']
export type AccountContext = paths['/api/v1/account/context']['get']['responses'][200]['content']['application/json']
export type AccountSessions = paths['/api/v1/account/sessions']['get']['responses'][200]['content']['application/json']
export type VerificationResponse = paths['/api/v1/account/verification']['post']['responses'][200]['content']['application/json']
export type LoginRequest = paths['/api/v1/account/login']['post']['requestBody']['content']['application/json']
export type VerificationRequest = paths['/api/v1/account/verification']['post']['requestBody']['content']['application/json']
export type RegistrationRequest = paths['/api/v1/account/register']['post']['requestBody']['content']['application/json']
