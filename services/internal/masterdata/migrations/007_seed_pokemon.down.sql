 No newline at end of file                                                                                                                                              
-- Remove Type Matchups                                                                                                                                                 
DELETE FROM type_matchup WHERE attacking_type IN ('static', 'dynamic', 'functional', 'procedural');                                                                     
                                                                                                                                                                        
-- Remove Pokemon (Programming Languages)                                                                                                                        
DELETE FROM pokemon WHERE id IN (                                                                                                                                
    '00000000-0000-0000-0000-000000000001',                                                                                                                      
    '00000000-0000-0000-0000-000000000002',                                                                                                                      
    '00000000-0000-0000-0000-000000000003',                                                                                                                      
    '00000000-0000-0000-0000-000000000005',                                                                                                                      
    '00000000-0000-0000-0000-000000000006',                                                                                                                      
    '00000000-0000-0000-0000-000000000007',                                                                                                                      
    '00000000-0000-0000-0000-000000000008',                                                                                                                      
    '00000000-0000-0000-0000-000000000009'                                                                                                                       
    ); 