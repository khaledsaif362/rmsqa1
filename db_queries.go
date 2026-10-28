package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const dbQueryMaxRows = 100

type dbLibraryQuery struct {
	ID         string
	Table      string
	Column     string
	NeedsValue bool
}

var dbLibraryQueries = map[string]dbLibraryQuery{
	"preview_ban_net_position":                                    {ID: "preview_ban_net_position", Table: "BAN_NET_POSITION", Column: "", NeedsValue: false},
	"preview_bhav_copy":                                           {ID: "preview_bhav_copy", Table: "BHAV_COPY", Column: "", NeedsValue: false},
	"preview_bse_gsm_temp":                                        {ID: "preview_bse_gsm_temp", Table: "BSE_GSM_TEMP", Column: "", NeedsValue: false},
	"preview_clientwise_mtf_exposer_limit":                        {ID: "preview_clientwise_mtf_exposer_limit", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "", NeedsValue: false},
	"preview_comm_orders":                                         {ID: "preview_comm_orders", Table: "COMM_ORDERS", Column: "", NeedsValue: false},
	"preview_drv_bhav_copy":                                       {ID: "preview_drv_bhav_copy", Table: "DRV_BHAV_COPY", Column: "", NeedsValue: false},
	"preview_drv_l1_watch":                                        {ID: "preview_drv_l1_watch", Table: "DRV_L1_WATCH", Column: "", NeedsValue: false},
	"preview_drv_orders":                                          {ID: "preview_drv_orders", Table: "DRV_ORDERS", Column: "", NeedsValue: false},
	"preview_entity_master":                                       {ID: "preview_entity_master", Table: "ENTITY_MASTER", Column: "", NeedsValue: false},
	"preview_eq_l1_watch":                                         {ID: "preview_eq_l1_watch", Table: "EQ_L1_WATCH", Column: "", NeedsValue: false},
	"preview_eq_orders":                                           {ID: "preview_eq_orders", Table: "EQ_ORDERS", Column: "", NeedsValue: false},
	"preview_exch_administration_master":                          {ID: "preview_exch_administration_master", Table: "EXCH_ADMINISTRATION_MASTER", Column: "", NeedsValue: false},
	"preview_file_error_log":                                      {ID: "preview_file_error_log", Table: "FILE_ERROR_LOG", Column: "", NeedsValue: false},
	"preview_go_client_allowed_immem":                             {ID: "preview_go_client_allowed_immem", Table: "GO_CLIENT_ALLOWED_IMMEM", Column: "", NeedsValue: false},
	"preview_go_client_mail_info":                                 {ID: "preview_go_client_mail_info", Table: "GO_CLIENT_MAIL_INFO", Column: "", NeedsValue: false},
	"preview_go_fileupload_config":                                {ID: "preview_go_fileupload_config", Table: "GO_FILEUPLOAD_CONFIG", Column: "", NeedsValue: false},
	"preview_go_user_master":                                      {ID: "preview_go_user_master", Table: "GO_USER_MASTER", Column: "", NeedsValue: false},
	"preview_l1_watch_active":                                     {ID: "preview_l1_watch_active", Table: "L1_WATCH_ACTIVE", Column: "", NeedsValue: false},
	"preview_mcx_security_info":                                   {ID: "preview_mcx_security_info", Table: "MCX_SECURITY_INFO", Column: "", NeedsValue: false},
	"preview_mrv_orders":                                          {ID: "preview_mrv_orders", Table: "MRV_ORDERS", Column: "", NeedsValue: false},
	"preview_net_position_table":                                  {ID: "preview_net_position_table", Table: "NET_POSITION_TABLE", Column: "", NeedsValue: false},
	"preview_nse_gsm_temp":                                        {ID: "preview_nse_gsm_temp", Table: "NSE_GSM_TEMP", Column: "", NeedsValue: false},
	"preview_rms_approve_pledg_script":                            {ID: "preview_rms_approve_pledg_script", Table: "RMS_APPROVE_PLEDG_SCRIPT", Column: "", NeedsValue: false},
	"preview_rms_clientwise_uploaded_limit":                       {ID: "preview_rms_clientwise_uploaded_limit", Table: "RMS_CLIENTWISE_UPLOADED_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_client_fno_pos_limit_detail":                     {ID: "preview_rms_client_fno_pos_limit_detail", Table: "RMS_CLIENT_FNO_POS_LIMIT_DETAIL", Column: "", NeedsValue: false},
	"preview_rms_client_security_limit":                           {ID: "preview_rms_client_security_limit", Table: "RMS_CLIENT_SECURITY_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_early_payin":                                     {ID: "preview_rms_early_payin", Table: "RMS_EARLY_PAYIN", Column: "", NeedsValue: false},
	"preview_rms_eq_var_elm":                                      {ID: "preview_rms_eq_var_elm", Table: "RMS_EQ_VAR_ELM", Column: "", NeedsValue: false},
	"preview_rms_fno_delta":                                       {ID: "preview_rms_fno_delta", Table: "RMS_FNO_DELTA", Column: "", NeedsValue: false},
	"preview_rms_fno_span_margin":                                 {ID: "preview_rms_fno_span_margin", Table: "RMS_FNO_SPAN_MARGIN", Column: "", NeedsValue: false},
	"preview_rms_fund_limit":                                      {ID: "preview_rms_fund_limit", Table: "RMS_FUND_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_ipo_allotment_temp":                              {ID: "preview_rms_ipo_allotment_temp", Table: "RMS_IPO_ALLOTMENT_TEMP", Column: "", NeedsValue: false},
	"preview_rms_limit_config":                                    {ID: "preview_rms_limit_config", Table: "RMS_LIMIT_CONFIG", Column: "", NeedsValue: false},
	"preview_rms_mcx_exp_margin":                                  {ID: "preview_rms_mcx_exp_margin", Table: "RMS_MCX_EXP_MARGIN", Column: "", NeedsValue: false},
	"preview_rms_member_fno_pos_limit_detail":                     {ID: "preview_rms_member_fno_pos_limit_detail", Table: "RMS_MEMBER_FNO_POS_LIMIT_DETAIL", Column: "", NeedsValue: false},
	"preview_rms_memwise_uploaded_limit":                          {ID: "preview_rms_memwise_uploaded_limit", Table: "RMS_MEMWISE_UPLOADED_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_process_log":                                     {ID: "preview_rms_process_log", Table: "RMS_PROCESS_LOG", Column: "", NeedsValue: false},
	"preview_rms_risk_profile_det":                                {ID: "preview_rms_risk_profile_det", Table: "RMS_RISK_PROFILE_DET", Column: "", NeedsValue: false},
	"preview_rms_risk_profile_mast":                               {ID: "preview_rms_risk_profile_mast", Table: "RMS_RISK_PROFILE_MAST", Column: "", NeedsValue: false},
	"preview_rms_screen_column_config":                            {ID: "preview_rms_screen_column_config", Table: "RMS_SCREEN_COLUMN_CONFIG", Column: "", NeedsValue: false},
	"preview_rms_scrip_margin_basket":                             {ID: "preview_rms_scrip_margin_basket", Table: "RMS_SCRIP_MARGIN_BASKET", Column: "", NeedsValue: false},
	"preview_rms_scrip_margin_basket_fno":                         {ID: "preview_rms_scrip_margin_basket_fno", Table: "RMS_SCRIP_MARGIN_BASKET_FNO", Column: "", NeedsValue: false},
	"preview_security_master":                                     {ID: "preview_security_master", Table: "SECURITY_MASTER", Column: "", NeedsValue: false},
	"preview_security_rms_block_detail":                           {ID: "preview_security_rms_block_detail", Table: "SECURITY_RMS_BLOCK_DETAIL", Column: "", NeedsValue: false},
	"preview_sem_active":                                          {ID: "preview_sem_active", Table: "SEM_ACTIVE", Column: "", NeedsValue: false},
	"preview_surveillance_msg_details":                            {ID: "preview_surveillance_msg_details", Table: "SURVEILLANCE_MSG_DETAILS", Column: "", NeedsValue: false},
	"preview_sys_parameters":                                      {ID: "preview_sys_parameters", Table: "SYS_PARAMETERS", Column: "", NeedsValue: false},
	"preview_user_master":                                         {ID: "preview_user_master", Table: "USER_MASTER", Column: "", NeedsValue: false},
	"preview_view_client_limit":                                   {ID: "preview_view_client_limit", Table: "VIEW_CLIENT_LIMIT", Column: "", NeedsValue: false},
	"filter_001_ban_net_position_bnp_client_id":                   {ID: "filter_001_ban_net_position_bnp_client_id", Table: "BAN_NET_POSITION", Column: "BNP_CLIENT_ID", NeedsValue: true},
	"filter_002_bhav_copy_bc_scrip_code":                          {ID: "filter_002_bhav_copy_bc_scrip_code", Table: "BHAV_COPY", Column: "BC_SCRIP_CODE", NeedsValue: true},
	"filter_003_bse_gsm_temp_v_id_1":                              {ID: "filter_003_bse_gsm_temp_v_id_1", Table: "BSE_GSM_TEMP", Column: "V_ID_1", NeedsValue: true},
	"filter_004_clientwise_mtf_exposer_limit_cme_client_id":       {ID: "filter_004_clientwise_mtf_exposer_limit_cme_client_id", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "CME_CLIENT_ID", NeedsValue: true},
	"filter_005_comm_orders_comm_client_id":                       {ID: "filter_005_comm_orders_comm_client_id", Table: "COMM_ORDERS", Column: "COMM_CLIENT_ID", NeedsValue: true},
	"filter_006_drv_bhav_copy_dbc_scrip_code":                     {ID: "filter_006_drv_bhav_copy_dbc_scrip_code", Table: "DRV_BHAV_COPY", Column: "DBC_SCRIP_CODE", NeedsValue: true},
	"filter_007_drv_l1_watch_dl1_scrip_code":                      {ID: "filter_007_drv_l1_watch_dl1_scrip_code", Table: "DRV_L1_WATCH", Column: "DL1_SCRIP_CODE", NeedsValue: true},
	"filter_008_drv_orders_drv_client_id":                         {ID: "filter_008_drv_orders_drv_client_id", Table: "DRV_ORDERS", Column: "DRV_CLIENT_ID", NeedsValue: true},
	"filter_009_entity_master_entity_code":                        {ID: "filter_009_entity_master_entity_code", Table: "ENTITY_MASTER", Column: "ENTITY_CODE", NeedsValue: true},
	"filter_010_eq_l1_watch_l1_scrip_code":                        {ID: "filter_010_eq_l1_watch_l1_scrip_code", Table: "EQ_L1_WATCH", Column: "L1_SCRIP_CODE", NeedsValue: true},
	"filter_011_eq_orders_eq_client_id":                           {ID: "filter_011_eq_orders_eq_client_id", Table: "EQ_ORDERS", Column: "EQ_CLIENT_ID", NeedsValue: true},
	"filter_012_exch_administration_master_eam_broker_status":     {ID: "filter_012_exch_administration_master_eam_broker_status", Table: "EXCH_ADMINISTRATION_MASTER", Column: "EAM_BROKER_STATUS", NeedsValue: true},
	"filter_013_file_error_log_upload_time":                       {ID: "filter_013_file_error_log_upload_time", Table: "FILE_ERROR_LOG", Column: "UPLOAD_TIME", NeedsValue: true},
	"filter_014_go_client_allowed_immem_client_id":                {ID: "filter_014_go_client_allowed_immem_client_id", Table: "GO_CLIENT_ALLOWED_IMMEM", Column: "CLIENT_ID", NeedsValue: true},
	"filter_015_go_client_mail_info_g_client_id":                  {ID: "filter_015_go_client_mail_info_g_client_id", Table: "GO_CLIENT_MAIL_INFO", Column: "G_CLIENT_ID", NeedsValue: true},
	"filter_016_go_fileupload_config_date_flag":                   {ID: "filter_016_go_fileupload_config_date_flag", Table: "GO_FILEUPLOAD_CONFIG", Column: "DATE_FLAG", NeedsValue: true},
	"filter_017_go_user_master_user_id":                           {ID: "filter_017_go_user_master_user_id", Table: "GO_USER_MASTER", Column: "USER_ID", NeedsValue: true},
	"filter_018_l1_watch_active_l1_scrip_code":                    {ID: "filter_018_l1_watch_active_l1_scrip_code", Table: "L1_WATCH_ACTIVE", Column: "L1_SCRIP_CODE", NeedsValue: true},
	"filter_019_mcx_security_info_mcx_symbol":                     {ID: "filter_019_mcx_security_info_mcx_symbol", Table: "MCX_SECURITY_INFO", Column: "MCX_SYMBOL", NeedsValue: true},
	"filter_020_mrv_orders_mrv_client_id":                         {ID: "filter_020_mrv_orders_mrv_client_id", Table: "MRV_ORDERS", Column: "MRV_CLIENT_ID", NeedsValue: true},
	"filter_021_net_position_table_npt_client_id":                 {ID: "filter_021_net_position_table_npt_client_id", Table: "NET_POSITION_TABLE", Column: "NPT_CLIENT_ID", NeedsValue: true},
	"filter_022_nse_gsm_temp_v_id_1":                              {ID: "filter_022_nse_gsm_temp_v_id_1", Table: "NSE_GSM_TEMP", Column: "V_ID_1", NeedsValue: true},
	"filter_023_rms_approve_pledg_script_raps_isin":               {ID: "filter_023_rms_approve_pledg_script_raps_isin", Table: "RMS_APPROVE_PLEDG_SCRIPT", Column: "RAPS_ISIN", NeedsValue: true},
	"filter_024_rms_clientwise_uploaded_limit_rcul_symbol":        {ID: "filter_024_rms_clientwise_uploaded_limit_rcul_symbol", Table: "RMS_CLIENTWISE_UPLOADED_LIMIT", Column: "RCUL_SYMBOL", NeedsValue: true},
	"filter_025_rms_client_fno_pos_limit_detail_client_id":        {ID: "filter_025_rms_client_fno_pos_limit_detail_client_id", Table: "RMS_CLIENT_FNO_POS_LIMIT_DETAIL", Column: "CLIENT_ID", NeedsValue: true},
	"filter_026_rms_client_security_limit_client_id":              {ID: "filter_026_rms_client_security_limit_client_id", Table: "RMS_CLIENT_SECURITY_LIMIT", Column: "CLIENT_ID", NeedsValue: true},
	"filter_027_rms_early_payin_rep_client_id":                    {ID: "filter_027_rms_early_payin_rep_client_id", Table: "RMS_EARLY_PAYIN", Column: "REP_CLIENT_ID", NeedsValue: true},
	"filter_028_rms_eq_var_elm_sec_symbol":                        {ID: "filter_028_rms_eq_var_elm_sec_symbol", Table: "RMS_EQ_VAR_ELM", Column: "SEC_SYMBOL", NeedsValue: true},
	"filter_029_rms_fno_delta_rfd_scrip_code":                     {ID: "filter_029_rms_fno_delta_rfd_scrip_code", Table: "RMS_FNO_DELTA", Column: "RFD_SCRIP_CODE", NeedsValue: true},
	"filter_030_rms_fno_span_margin_rsm_security_id":              {ID: "filter_030_rms_fno_span_margin_rsm_security_id", Table: "RMS_FNO_SPAN_MARGIN", Column: "RSM_SECURITY_ID", NeedsValue: true},
	"filter_031_rms_fund_limit_client_id":                         {ID: "filter_031_rms_fund_limit_client_id", Table: "RMS_FUND_LIMIT", Column: "CLIENT_ID", NeedsValue: true},
	"filter_032_rms_ipo_allotment_temp_symbol":                    {ID: "filter_032_rms_ipo_allotment_temp_symbol", Table: "RMS_IPO_ALLOTMENT_TEMP", Column: "SYMBOL", NeedsValue: true},
	"filter_033_rms_limit_config_rlc_segment":                     {ID: "filter_033_rms_limit_config_rlc_segment", Table: "RMS_LIMIT_CONFIG", Column: "RLC_SEGMENT", NeedsValue: true},
	"filter_034_rms_mcx_exp_margin_rmem_security_id":              {ID: "filter_034_rms_mcx_exp_margin_rmem_security_id", Table: "RMS_MCX_EXP_MARGIN", Column: "RMEM_SECURITY_ID", NeedsValue: true},
	"filter_035_rms_member_fno_pos_limit_detail_exch_symbol":      {ID: "filter_035_rms_member_fno_pos_limit_detail_exch_symbol", Table: "RMS_MEMBER_FNO_POS_LIMIT_DETAIL", Column: "EXCH_SYMBOL", NeedsValue: true},
	"filter_036_rms_memwise_uploaded_limit_rmul_symbol":           {ID: "filter_036_rms_memwise_uploaded_limit_rmul_symbol", Table: "RMS_MEMWISE_UPLOADED_LIMIT", Column: "RMUL_SYMBOL", NeedsValue: true},
	"filter_037_rms_process_log_process_date":                     {ID: "filter_037_rms_process_log_process_date", Table: "RMS_PROCESS_LOG", Column: "PROCESS_DATE", NeedsValue: true},
	"filter_038_rms_risk_profile_det_rpd_scrip_code":              {ID: "filter_038_rms_risk_profile_det_rpd_scrip_code", Table: "RMS_RISK_PROFILE_DET", Column: "RPD_SCRIP_CODE", NeedsValue: true},
	"filter_039_rms_risk_profile_mast_rpm_product_enabled":        {ID: "filter_039_rms_risk_profile_mast_rpm_product_enabled", Table: "RMS_RISK_PROFILE_MAST", Column: "RPM_PRODUCT_ENABLED", NeedsValue: true},
	"filter_040_rms_screen_column_config_rscc_screen_name":        {ID: "filter_040_rms_screen_column_config_rscc_screen_name", Table: "RMS_SCREEN_COLUMN_CONFIG", Column: "RSCC_SCREEN_NAME", NeedsValue: true},
	"filter_041_rms_scrip_margin_basket_rsmb_scrip_code":          {ID: "filter_041_rms_scrip_margin_basket_rsmb_scrip_code", Table: "RMS_SCRIP_MARGIN_BASKET", Column: "RSMB_SCRIP_CODE", NeedsValue: true},
	"filter_042_rms_scrip_margin_basket_fno_rsmbf_scrip_code":     {ID: "filter_042_rms_scrip_margin_basket_fno_rsmbf_scrip_code", Table: "RMS_SCRIP_MARGIN_BASKET_FNO", Column: "RSMBF_SCRIP_CODE", NeedsValue: true},
	"filter_043_security_master_sm_scrip_code":                    {ID: "filter_043_security_master_sm_scrip_code", Table: "SECURITY_MASTER", Column: "SM_SCRIP_CODE", NeedsValue: true},
	"filter_044_security_rms_block_detail_client_id":              {ID: "filter_044_security_rms_block_detail_client_id", Table: "SECURITY_RMS_BLOCK_DETAIL", Column: "CLIENT_ID", NeedsValue: true},
	"filter_045_sem_active_sm_scrip_code":                         {ID: "filter_045_sem_active_sm_scrip_code", Table: "SEM_ACTIVE", Column: "SM_SCRIP_CODE", NeedsValue: true},
	"filter_046_surveillance_msg_details_smd_client_id":           {ID: "filter_046_surveillance_msg_details_smd_client_id", Table: "SURVEILLANCE_MSG_DETAILS", Column: "SMD_CLIENT_ID", NeedsValue: true},
	"filter_047_sys_parameters_param_name":                        {ID: "filter_047_sys_parameters_param_name", Table: "SYS_PARAMETERS", Column: "PARAM_NAME", NeedsValue: true},
	"filter_048_user_master_user_entity_code":                     {ID: "filter_048_user_master_user_entity_code", Table: "USER_MASTER", Column: "USER_ENTITY_CODE", NeedsValue: true},
	"filter_049_view_client_limit_client_id":                      {ID: "filter_049_view_client_limit_client_id", Table: "VIEW_CLIENT_LIMIT", Column: "CLIENT_ID", NeedsValue: true},
	"filter_050_ban_net_position_bnp_underlying_scrip_code":       {ID: "filter_050_ban_net_position_bnp_underlying_scrip_code", Table: "BAN_NET_POSITION", Column: "BNP_UNDERLYING_SCRIP_CODE", NeedsValue: true},
	"filter_051_bhav_copy_bc_segment":                             {ID: "filter_051_bhav_copy_bc_segment", Table: "BHAV_COPY", Column: "BC_SEGMENT", NeedsValue: true},
	"filter_052_bse_gsm_temp_v_id_2":                              {ID: "filter_052_bse_gsm_temp_v_id_2", Table: "BSE_GSM_TEMP", Column: "V_ID_2", NeedsValue: true},
	"filter_053_clientwise_mtf_exposer_limit_cme_symbol":          {ID: "filter_053_clientwise_mtf_exposer_limit_cme_symbol", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "CME_SYMBOL", NeedsValue: true},
	"filter_054_comm_orders_comm_scrip_code":                      {ID: "filter_054_comm_orders_comm_scrip_code", Table: "COMM_ORDERS", Column: "COMM_SCRIP_CODE", NeedsValue: true},
	"filter_055_drv_bhav_copy_dbc_segment":                        {ID: "filter_055_drv_bhav_copy_dbc_segment", Table: "DRV_BHAV_COPY", Column: "DBC_SEGMENT", NeedsValue: true},
	"filter_056_drv_l1_watch_dl1_exch_scrip_code":                 {ID: "filter_056_drv_l1_watch_dl1_exch_scrip_code", Table: "DRV_L1_WATCH", Column: "DL1_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_057_drv_orders_drv_scrip_code":                        {ID: "filter_057_drv_orders_drv_scrip_code", Table: "DRV_ORDERS", Column: "DRV_SCRIP_CODE", NeedsValue: true},
	"filter_058_entity_master_entity_risk_profile":                {ID: "filter_058_entity_master_entity_risk_profile", Table: "ENTITY_MASTER", Column: "ENTITY_RISK_PROFILE", NeedsValue: true},
	"filter_059_eq_l1_watch_l1_exch_scrip_code":                   {ID: "filter_059_eq_l1_watch_l1_exch_scrip_code", Table: "EQ_L1_WATCH", Column: "L1_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_060_eq_orders_eq_scrip_code":                          {ID: "filter_060_eq_orders_eq_scrip_code", Table: "EQ_ORDERS", Column: "EQ_SCRIP_CODE", NeedsValue: true},
	"filter_061_exch_administration_master_eam_logon_status":      {ID: "filter_061_exch_administration_master_eam_logon_status", Table: "EXCH_ADMINISTRATION_MASTER", Column: "EAM_LOGON_STATUS", NeedsValue: true},
	"filter_062_file_error_log_err_code":                          {ID: "filter_062_file_error_log_err_code", Table: "FILE_ERROR_LOG", Column: "ERR_CODE", NeedsValue: true},
	"filter_063_go_client_allowed_immem_segment":                  {ID: "filter_063_go_client_allowed_immem_segment", Table: "GO_CLIENT_ALLOWED_IMMEM", Column: "SEGMENT", NeedsValue: true},
	"filter_064_go_client_mail_info_g_email_id":                   {ID: "filter_064_go_client_mail_info_g_email_id", Table: "GO_CLIENT_MAIL_INFO", Column: "G_EMAIL_ID", NeedsValue: true},
	"filter_065_go_fileupload_config_priority":                    {ID: "filter_065_go_fileupload_config_priority", Table: "GO_FILEUPLOAD_CONFIG", Column: "PRIORITY", NeedsValue: true},
	"filter_066_go_user_master_last_update_time":                  {ID: "filter_066_go_user_master_last_update_time", Table: "GO_USER_MASTER", Column: "LAST_UPDATE_TIME", NeedsValue: true},
	"filter_067_l1_watch_active_l1_exch_scrip_code":               {ID: "filter_067_l1_watch_active_l1_exch_scrip_code", Table: "L1_WATCH_ACTIVE", Column: "L1_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_068_mcx_security_info_mcx_expiry_process_date":        {ID: "filter_068_mcx_security_info_mcx_expiry_process_date", Table: "MCX_SECURITY_INFO", Column: "MCX_EXPIRY_PROCESS_DATE", NeedsValue: true},
	"filter_069_mrv_orders_mrv_scrip_code":                        {ID: "filter_069_mrv_orders_mrv_scrip_code", Table: "MRV_ORDERS", Column: "MRV_SCRIP_CODE", NeedsValue: true},
	"filter_070_net_position_table_npt_security_id":               {ID: "filter_070_net_position_table_npt_security_id", Table: "NET_POSITION_TABLE", Column: "NPT_SECURITY_ID", NeedsValue: true},
	"filter_071_nse_gsm_temp_v_id_2":                              {ID: "filter_071_nse_gsm_temp_v_id_2", Table: "NSE_GSM_TEMP", Column: "V_ID_2", NeedsValue: true},
	"filter_072_rms_clientwise_uploaded_limit_rcul_exch_id":       {ID: "filter_072_rms_clientwise_uploaded_limit_rcul_exch_id", Table: "RMS_CLIENTWISE_UPLOADED_LIMIT", Column: "RCUL_EXCH_ID", NeedsValue: true},
	"filter_073_rms_client_fno_pos_limit_detail_exch_symbol":      {ID: "filter_073_rms_client_fno_pos_limit_detail_exch_symbol", Table: "RMS_CLIENT_FNO_POS_LIMIT_DETAIL", Column: "EXCH_SYMBOL", NeedsValue: true},
	"filter_074_rms_client_security_limit_nse_scrip_code":         {ID: "filter_074_rms_client_security_limit_nse_scrip_code", Table: "RMS_CLIENT_SECURITY_LIMIT", Column: "NSE_SCRIP_CODE", NeedsValue: true},
	"filter_075_rms_early_payin_rep_isin_reqid":                   {ID: "filter_075_rms_early_payin_rep_isin_reqid", Table: "RMS_EARLY_PAYIN", Column: "REP_ISIN_REQID", NeedsValue: true},
	"filter_076_rms_eq_var_elm_isin":                              {ID: "filter_076_rms_eq_var_elm_isin", Table: "RMS_EQ_VAR_ELM", Column: "ISIN", NeedsValue: true},
	"filter_077_rms_fno_delta_rfd_underlying_scrip_code":          {ID: "filter_077_rms_fno_delta_rfd_underlying_scrip_code", Table: "RMS_FNO_DELTA", Column: "RFD_UNDERLYING_SCRIP_CODE", NeedsValue: true},
	"filter_078_rms_fno_span_margin_rsm_symbol_name":              {ID: "filter_078_rms_fno_span_margin_rsm_symbol_name", Table: "RMS_FNO_SPAN_MARGIN", Column: "RSM_SYMBOL_NAME", NeedsValue: true},
	"filter_079_rms_fund_limit_segment":                           {ID: "filter_079_rms_fund_limit_segment", Table: "RMS_FUND_LIMIT", Column: "SEGMENT", NeedsValue: true},
	"filter_080_rms_ipo_allotment_temp_upi_last_status":           {ID: "filter_080_rms_ipo_allotment_temp_upi_last_status", Table: "RMS_IPO_ALLOTMENT_TEMP", Column: "UPI_LAST_STATUS", NeedsValue: true},
	"filter_081_rms_limit_config_rcl_segment_include":             {ID: "filter_081_rms_limit_config_rcl_segment_include", Table: "RMS_LIMIT_CONFIG", Column: "RCL_SEGMENT_INCLUDE", NeedsValue: true},
	"filter_082_rms_mcx_exp_margin_rmem_symbol_name":              {ID: "filter_082_rms_mcx_exp_margin_rmem_symbol_name", Table: "RMS_MCX_EXP_MARGIN", Column: "RMEM_SYMBOL_NAME", NeedsValue: true},
	"filter_083_rms_member_fno_pos_limit_detail_tot_gross_qty":    {ID: "filter_083_rms_member_fno_pos_limit_detail_tot_gross_qty", Table: "RMS_MEMBER_FNO_POS_LIMIT_DETAIL", Column: "TOT_GROSS_QTY", NeedsValue: true},
	"filter_084_rms_memwise_uploaded_limit_rmul_prop_allowed_qty": {ID: "filter_084_rms_memwise_uploaded_limit_rmul_prop_allowed_qty", Table: "RMS_MEMWISE_UPLOADED_LIMIT", Column: "RMUL_PROP_ALLOWED_QTY", NeedsValue: true},
	"filter_085_rms_process_log_upload_file_name":                 {ID: "filter_085_rms_process_log_upload_file_name", Table: "RMS_PROCESS_LOG", Column: "UPLOAD_FILE_NAME", NeedsValue: true},
	"filter_086_rms_risk_profile_det_rpd_symbol":                  {ID: "filter_086_rms_risk_profile_det_rpd_symbol", Table: "RMS_RISK_PROFILE_DET", Column: "RPD_SYMBOL", NeedsValue: true},
	"filter_087_rms_risk_profile_mast_product_ql_check":           {ID: "filter_087_rms_risk_profile_mast_product_ql_check", Table: "RMS_RISK_PROFILE_MAST", Column: "PRODUCT_QL_CHECK", NeedsValue: true},
	"filter_088_rms_screen_column_config_rscc_column_names":       {ID: "filter_088_rms_screen_column_config_rscc_column_names", Table: "RMS_SCREEN_COLUMN_CONFIG", Column: "RSCC_COLUMN_NAMES", NeedsValue: true},
	"filter_089_rms_scrip_margin_basket_rsmb_segment":             {ID: "filter_089_rms_scrip_margin_basket_rsmb_segment", Table: "RMS_SCRIP_MARGIN_BASKET", Column: "RSMB_SEGMENT", NeedsValue: true},
	"filter_090_rms_scrip_margin_basket_fno_rsmbf_segment":        {ID: "filter_090_rms_scrip_margin_basket_fno_rsmbf_segment", Table: "RMS_SCRIP_MARGIN_BASKET_FNO", Column: "RSMBF_SEGMENT", NeedsValue: true},
	"filter_091_security_master_sm_exch_scrip_code":               {ID: "filter_091_security_master_sm_exch_scrip_code", Table: "SECURITY_MASTER", Column: "SM_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_092_security_rms_block_detail_scrip_code":             {ID: "filter_092_security_rms_block_detail_scrip_code", Table: "SECURITY_RMS_BLOCK_DETAIL", Column: "SCRIP_CODE", NeedsValue: true},
	"filter_093_sem_active_sm_exch_scrip_code":                    {ID: "filter_093_sem_active_sm_exch_scrip_code", Table: "SEM_ACTIVE", Column: "SM_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_094_surveillance_msg_details_smd_scrip_code":          {ID: "filter_094_surveillance_msg_details_smd_scrip_code", Table: "SURVEILLANCE_MSG_DETAILS", Column: "SMD_SCRIP_CODE", NeedsValue: true},
	"filter_095_sys_parameters_param_value":                       {ID: "filter_095_sys_parameters_param_value", Table: "SYS_PARAMETERS", Column: "PARAM_VALUE", NeedsValue: true},
	"filter_096_user_master_user_status":                          {ID: "filter_096_user_master_user_status", Table: "USER_MASTER", Column: "USER_STATUS", NeedsValue: true},
	"filter_097_view_client_limit_segment":                        {ID: "filter_097_view_client_limit_segment", Table: "VIEW_CLIENT_LIMIT", Column: "SEGMENT", NeedsValue: true},
	"filter_098_ban_net_position_bnp_security_id":                 {ID: "filter_098_ban_net_position_bnp_security_id", Table: "BAN_NET_POSITION", Column: "BNP_SECURITY_ID", NeedsValue: true},
	"filter_099_bhav_copy_bc_date":                                {ID: "filter_099_bhav_copy_bc_date", Table: "BHAV_COPY", Column: "BC_DATE", NeedsValue: true},
	"filter_100_bse_gsm_temp_v_id_3":                              {ID: "filter_100_bse_gsm_temp_v_id_3", Table: "BSE_GSM_TEMP", Column: "V_ID_3", NeedsValue: true},
	"filter_101_clientwise_mtf_exposer_limit_cme_tot_qty":         {ID: "filter_101_clientwise_mtf_exposer_limit_cme_tot_qty", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "CME_TOT_QTY", NeedsValue: true},
}

// dbQueryEscape escapes a user-supplied value; SQL identifiers are never user-supplied.
func dbQueryEscape(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, "'", "''")
}

func dbQueryQuote(value string) string { return "'" + dbQueryEscape(value) + "'" }

// dbQueryBuildSQL only builds SQL from the fixed allowlist above. Every query has LIMIT 100.
func dbQueryBuildSQL(id, value string) (string, error) {
	q, ok := dbLibraryQueries[id]
	if !ok {
		return "", fmt.Errorf("Unknown predefined query")
	}
	table := "`" + q.Table + "`"
	if !q.NeedsValue {
		return "SELECT * FROM " + table + " LIMIT 100", nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("Enter a value for " + q.Column)
	}
	if len(value) > 100 {
		return "", fmt.Errorf("Parameter too long (maximum 100 characters)")
	}
	column := "`" + q.Column + "`"
	return "SELECT * FROM " + table + " WHERE " + column + " = " + dbQueryQuote(value) + " LIMIT 100", nil
}

// apiDBQueryLibrary executes only allowlisted read-only SELECT queries.
func apiDBQueryLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		dbQueryJSONError(w, http.StatusMethodNotAllowed, "Use POST")
		return
	}
	if err := dbQueryLoadEnv(); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Cannot read database configuration")
		return
	}
	sess, ok := dbQueryCurrentSession(w, r)
	if !ok {
		return
	}
	var req struct {
		DB    string `json:"db"`
		ID    string `json:"id"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.DB != sess.Server {
		dbQueryJSONError(w, http.StatusForbidden, "Selected server does not match connected session")
		return
	}
	query, err := dbQueryBuildSQL(req.ID, req.Value)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	host, port, name, user, pass, err := dbQueryConfig(sess.Server)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mysql", "--batch", "--raw", "--connect-timeout=5", "-h", host, "-P", port, "-u", user, name, "-e", query)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		dbQueryJSONError(w, http.StatusBadGateway, "Query failed; verify table, columns, permissions and MySQL client")
		return
	}
	out := strings.TrimSpace(string(output))
	if out == "" {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"columns": []string{}, "rows": []interface{}{}, "count": 0, "max_rows": dbQueryMaxRows})
		return
	}
	lines := strings.Split(out, "\n")
	columns := strings.Split(lines[0], "\t")
	rows := make([]map[string]interface{}, 0, minInt(len(lines)-1, dbQueryMaxRows))
	for _, line := range lines[1:] {
		if len(rows) >= dbQueryMaxRows {
			break
		}
		fields := strings.Split(line, "\t")
		if len(fields) != len(columns) {
			continue
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			if fields[i] == "NULL" {
				row[col] = nil
			} else {
				row[col] = fields[i]
			}
		}
		rows = append(rows, row)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"columns": columns, "rows": rows, "count": len(rows), "max_rows": dbQueryMaxRows})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
